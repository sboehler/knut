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

package register

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"time"

	"github.com/sboehler/knut/cmd/flags"
	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/common/date"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast/parser"
	"github.com/sboehler/knut/lib/journal/process"

	"github.com/spf13/cobra"
)

// CreateCmd creates the command.
func CreateCmd() *cobra.Command {

	var r runner

	// Cmd is the balance command.
	var c = &cobra.Command{
		Use:   "register",
		Short: "create a register journal",
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
	sortAlphabetically                      bool
	digits                                  int32
	accounts, commodities                   flags.RegexFlag
	interval                                flags.IntervalFlags
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
	c.Flags().BoolVarP(&r.sortAlphabetically, "sort", "a", false, "Sort accounts alphabetically")
	c.Flags().BoolVarP(&r.showCommodities, "show-commodities", "s", false, "Show commodities on their own rows")
	r.interval.Setup(c.Flags())
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
		interval  date.Interval

		err error
	)
	if time.Time(r.to).IsZero() {
		r.to = flags.DateFlag(date.Today())
	}
	if valuation, err = r.valuation.Value(jctx); err != nil {
		return err
	}
	if interval == date.Once {
		interval = date.Daily
	}

	var (
		par = parser.RecursiveParser{
			File:    args[0],
			Context: jctx,
		}
		astBuilder = process.ASTBuilder{
			Context: jctx,
		}
		astExpander = process.ASTExpander{
			Expand: true,
			Filter: journal.Filter{
				Accounts:    r.accounts.Value(),
				Commodities: r.commodities.Value(),
			},
		}
		pastBuilder = process.PASTBuilder{
			Context: jctx,
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
			From:     r.from.Value(),
			To:       r.to.Value(),
			Interval: interval,
			Last:     r.last,
		}
		ctx = cmd.Context()
	)

	ch0, errCh0 := par.Parse(ctx)
	ch1, errCh1 := astBuilder.BuildAST(ctx, ch0)
	ch2, errCh2 := astExpander.ExpandAndFilterAST(ctx, ch1)
	ch3, errCh3 := pastBuilder.ProcessAST(ctx, ch2)
	ch4, errCh4 := priceUpdater.ProcessStream(ctx, ch3)
	ch5, errCh5 := valuator.ProcessStream(ctx, ch4)
	ch6, errCh6 := periodFilter.ProcessStream(ctx, ch5)

	resCh := ch6

	errCh := cpr.Demultiplex(errCh0, errCh1, errCh2, errCh3, errCh4, errCh5, errCh6)

	_, ok, err := cpr.Get(resCh, errCh)
	if !ok {
		return fmt.Errorf("no report was produced")
	}
	if err != nil {
		return err
	}
	out := bufio.NewWriter(cmd.OutOrStdout())
	defer out.Flush()
	_, err = out.WriteString("register")
	return err
}
