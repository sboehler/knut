// Copyright 2021 Silvio Böhler
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package balance

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"sync"
	"time"

	"github.com/sboehler/knut/cmd/flags"
	"github.com/sboehler/knut/lib/balance/report"
	"github.com/sboehler/knut/lib/common/date"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/past/process"
	"github.com/sboehler/knut/lib/table"

	"github.com/spf13/cobra"
)

// CreateCmd creates the command.
func CreateCmd() *cobra.Command {

	var r runner

	// Cmd is the balance command.
	var c = &cobra.Command{
		Use:   "balance",
		Short: "create a balance sheet",
		Long:  `Compute a balance for a date or set of dates.`,
		Args:  cobra.ExactValidArgs(1),
		Run:   r.run,
	}
	r.setupFlags(c)
	return c
}

type runner struct {
	cpuprofile                              string
	from, to                                flags.DateFlag
	last                                    int
	diff, showCommodities, thousands, color bool
	digits                                  int32
	accounts, commodities                   flags.RegexFlag
	period                                  flags.PeriodFlags
	mapping                                 flags.MappingFlag
	valuation                               flags.CommodityFlag
}

func (r *runner) run(cmd *cobra.Command, args []string) {
	if r.cpuprofile != "" {
		f, err := os.Create(r.cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	if err := r.execute(cmd, args); err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), err)
		os.Exit(1)
	}
}

func (r *runner) setupFlags(c *cobra.Command) {
	c.Flags().StringVar(&r.cpuprofile, "cpuprofile", "", "file to write profile")
	c.Flags().Var(&r.from, "from", "from date")
	c.Flags().Var(&r.to, "to", "to date")
	c.Flags().IntVar(&r.last, "last", 0, "last n periods")
	c.Flags().BoolVarP(&r.diff, "diff", "d", false, "diff")
	c.Flags().BoolVarP(&r.showCommodities, "show-commodities", "s", false, "Show commodities on their own rows")
	r.period.Setup(c.Flags())
	c.Flags().VarP(&r.valuation, "val", "v", "valuate in the given commodity")
	c.Flags().VarP(&r.mapping, "map", "m", "<level>,<regex>")
	c.Flags().Var(&r.accounts, "account", "filter accounts with a regex")
	c.Flags().Var(&r.commodities, "commodity", "filter commodities with a regex")
	c.Flags().Int32Var(&r.digits, "digits", 0, "round to number of digits")
	c.Flags().BoolVarP(&r.thousands, "thousands", "k", false, "show numbers in units of 1000")
	c.Flags().BoolVar(&r.color, "color", false, "print output in color")
}

func (r runner) execute(cmd *cobra.Command, args []string) error {
	var (
		jctx = journal.NewContext()

		valuation *journal.Commodity
		period    date.Period

		err error
	)
	if time.Time(r.to).IsZero() {
		r.to = flags.DateFlag(date.Today())
	}
	if valuation, err = r.valuation.Value(jctx); err != nil {
		return err
	}
	if period, err = r.period.Value(); err != nil {
		return err
	}

	var (
		filter = journal.Filter{
			Accounts:    r.accounts.Value(),
			Commodities: r.commodities.Value(),
		}
		astBuilder = process.ASTBuilder{
			Context: jctx,
		}
		pastBuilder = process.PASTBuilder{
			Context: jctx,
			Filter:  filter,
			Expand:  true,
		}
		priceUpdater = process.PriceUpdater{
			Context:   jctx,
			Valuation: valuation,
		}
		valuator = process.Valuator{
			Context:   jctx,
			Valuation: valuation,
		}
		periodFilter = process.PeriodFilter{
			From:   r.from.Value(),
			To:     r.to.Value(),
			Period: period,
			Last:   r.last,
			Diff:   r.diff,
		}
		differ = process.Differ{
			Diff: r.diff,
		}
		reportBuilder = report.Builder{
			Mapping: r.mapping.Value(),
		}
		reportRenderer = report.Renderer{
			Context:         jctx,
			ShowCommodities: r.showCommodities || valuation == nil,
		}
		tableRenderer = table.TextRenderer{
			Color:     r.color,
			Thousands: r.thousands,
			Round:     r.digits,
		}
	)
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	as, err := astBuilder.ASTFromPath(args[0])
	if err != nil {
		return err
	}

	ch1, errCh1 := pastBuilder.StreamFromAST(ctx, as)
	ch2, errCh2 := priceUpdater.ProcessStream(ctx, ch1)
	ch3, errCh3 := valuator.ProcessStream(ctx, ch2)
	ch4, errCh4 := periodFilter.ProcessStream(ctx, ch3)
	ch5, errCh5 := differ.ProcessStream(ctx, ch4)
	ch6, errCh6 := reportBuilder.FromStream(ctx, ch5)

	errCh := mergeErrors(errCh1, errCh2, errCh3, errCh4, errCh5, errCh6)

	for {
		select {

		case rep, ok := <-ch6:
			if !ok {
				return fmt.Errorf("no report was produced")
			}
			var out = bufio.NewWriter(cmd.OutOrStdout())
			defer out.Flush()
			return tableRenderer.Render(reportRenderer.Render(rep), out)

		case err, ok := <-errCh:
			if !ok {
				errCh = nil
			} else if err != nil {
				return err
			}
		}
	}
}

func mergeErrors(inChs ...<-chan error) chan error {
	var (
		wg    sync.WaitGroup
		errCh = make(chan error)
	)
	wg.Add(len(inChs))
	for _, inCh := range inChs {
		go func(ch <-chan error) {
			defer wg.Done()
			for err := range ch {
				errCh <- err
			}
		}(inCh)
	}
	go func() {
		wg.Wait()
		close(errCh)
	}()
	return errCh
}
