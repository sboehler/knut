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

package asyncbalance

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
	"github.com/sboehler/knut/lib/balance"
	"github.com/sboehler/knut/lib/date"
	"github.com/sboehler/knut/lib/ledger"
	"github.com/sboehler/knut/lib/parser"
	"github.com/sboehler/knut/lib/report"
	"github.com/sboehler/knut/lib/table"

	"github.com/spf13/cobra"
)

// CreateCmd creates the command.
func CreateCmd() *cobra.Command {

	var r runner

	// Cmd is the balance command.
	var c = &cobra.Command{
		Use:   "asyncbalance",
		Short: "create a balance sheet",
		Long:  `Compute a balance for a date or set of dates.`,

		Args: cobra.ExactValidArgs(1),

		Hidden: true,

		Run: r.run,
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
		ctx = ledger.NewContext()

		valuation *ledger.Commodity
		period    date.Period

		err error
	)
	if time.Time(r.to).IsZero() {
		now := time.Now()
		r.to = flags.DateFlag(time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC))
	}
	if valuation, err = r.valuation.Value(ctx); err != nil {
		return err
	}
	if period, err = r.period.Value(); err != nil {
		return err
	}

	var (
		parser = parser.RecursiveParser{
			File:    args[0],
			Context: ctx,
		}
		bal    = balance.New(ctx, valuation)
		filter = ledger.Filter{
			Accounts:    r.accounts.Value(),
			Commodities: r.commodities.Value(),
		}
		rep = &report.Report{
			Value:   valuation != nil,
			Mapping: r.mapping.Value(),
		}
		reportRenderer = report.Renderer{
			Context:         ctx,
			ShowCommodities: r.showCommodities || valuation == nil,
			Report:          rep,
		}
		tableRenderer = table.TextRenderer{
			Color:     r.color,
			Thousands: r.thousands,
			Round:     r.digits,
		}
	)

	var l ledger.Ledger
	if l, err = parser.BuildLedger(filter); err != nil {
		return err
	}

	var (
		balCh        = make(chan *balance.Balance)
		cotx, cancel = context.WithCancel(context.Background())
	)
	defer cancel()

	b1, err1Ch := balance.PreStage(cotx, l, balCh)
	b2 := balance.UpdatePrices(cotx, l, valuation, b1)
	b3, err3Ch := balance.PostStage(cotx, l, b2)
	snapshots, b4, err4Ch := balance.Snapshot(cotx, balance.SnapshotConfig{
		Last:   r.last,
		From:   r.from.Value(),
		Diff:   r.diff,
		Period: period,
		To:     r.to.Value(),
	}, l, b3)

	var errcList = []<-chan error{err1Ch, err3Ch, err4Ch}

	go func() {
		defer close(balCh)
		balCh <- bal
		for b := range b4 {
			balCh <- b
		}
	}()

	errs := merge(errcList...)

	var out = bufio.NewWriter(cmd.OutOrStdout())
	defer out.Flush()

	for {
		select {
		case bal, ok := <-snapshots:
			if !ok {
				snapshots = nil
			} else {
				rep.Add(bal)
			}
		case err, ok := <-errs:
			if !ok {
				errs = nil
			} else {
				if err != nil {
					return err
				}
			}
		}
		if snapshots == nil && errs == nil {
			return tableRenderer.Render(reportRenderer.Render(), out)
		}

	}
}

func merge(cs ...<-chan error) <-chan error {
	var wg sync.WaitGroup
	out := make(chan error)

	// Start an output goroutine for each input channel in cs.  output
	// copies values from c to out until c is closed, then calls wg.Done.
	output := func(c <-chan error) {
		for n := range c {
			out <- n
		}
		wg.Done()
	}
	wg.Add(len(cs))
	for _, c := range cs {
		go output(c)
	}

	// Start a goroutine to close out once all the output goroutines are
	// done.  This must start after the wg.Add call.
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}
