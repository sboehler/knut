// Copyright 2020 Silvio Böhler
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

package portfolio

import (
	"fmt"
	"log"
	"os"
	"runtime/pprof"

	"github.com/sboehler/knut/cmd/flags"
	"github.com/sboehler/knut/lib/balance"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
	"github.com/sboehler/knut/lib/journal/ast/parser"
	"github.com/sboehler/knut/lib/performance"

	"github.com/spf13/cobra"
)

// CreateCmd creates the command.
func CreateCmd() *cobra.Command {

	var r runner
	// Cmd is the balance command.
	var c = &cobra.Command{
		Use:   "portfolio",
		Short: "compute portfolio returns",
		Long:  `Compute portfolio returns.`,

		Args: cobra.ExactValidArgs(1),

		Hidden: true,

		Run: r.run,
	}
	r.setupFlags(c)
	return c
}

type runner struct {
	cpuprofile            string
	valuation             flags.CommodityFlag
	accounts, commodities flags.RegexFlag
}

func (r *runner) setupFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&r.cpuprofile, "cpuprofile", "", "file to write profile")
	cmd.Flags().VarP(&r.valuation, "val", "v", "valuate in the given commodity")
	cmd.Flags().Var(&r.accounts, "account", "filter accounts with a regex")
	cmd.Flags().Var(&r.commodities, "commodity", "filter commodities with a regex")
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

func (r *runner) execute(cmd *cobra.Command, args []string) error {
	var (
		ctx       = journal.NewContext()
		valuation *journal.Commodity
		err       error
	)
	if valuation, err = r.valuation.Value(ctx); err != nil {
		return err
	}

	var (
		p = parser.RecursiveParser{
			File:    args[0],
			Context: ctx,
		}
		bal    = balance.New(ctx, valuation)
		filter = journal.Filter{
			Commodities: r.commodities.Value(),
			Accounts:    r.accounts.Value(),
		}
		res   = new(performance.DailyPerfValues)
		steps = []ast.Processor{
			balance.DateUpdater{Balance: bal},
			balance.AccountOpener{Balance: bal},
			balance.TransactionBooker{Balance: bal},
			balance.ValueBooker{Balance: bal},
			balance.Asserter{Balance: bal},
			&balance.PriceUpdater{Balance: bal},
			balance.TransactionValuator{Balance: bal},
			balance.ValuationTransactionComputer{Balance: bal},
			balance.AccountCloser{Balance: bal},
			&performance.Valuator{Filter: filter, Result: res},
			&performance.FlowComputer{Filter: filter, Result: res},
			//TODO: compute performance here
		}
		perfCalc = performance.Calculator{
			Filter:    filter,
			Valuation: valuation,
		}
		l *ast.PAST
	)
	if l, err = p.BuildLedger(journal.Filter{}); err != nil {
		return err
	}
	if err = l.Process(steps); err != nil {
		return err
	}
	for range perfCalc.Perf(l) {
	}
	return nil
}
