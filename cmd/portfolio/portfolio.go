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
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"runtime/pprof"

	"github.com/sboehler/knut/cmd/flags"
	"github.com/sboehler/knut/lib/balance"
	"github.com/sboehler/knut/lib/ledger"
	"github.com/sboehler/knut/lib/parser"
	"github.com/sboehler/knut/lib/performance"

	"github.com/spf13/cobra"
)

// CreateCmd creates the command.
func CreateCmd() *cobra.Command {

	// Cmd is the balance command.
	var c = &cobra.Command{
		Use:   "portfolio",
		Short: "compute portfolio returns",
		Long:  `Compute portfolio returns.`,

		Args: cobra.ExactValidArgs(1),

		Run: run,
	}
	c.Flags().String("from", "", "from date")
	c.Flags().String("cpuprofile", "", "file to write profile")
	c.Flags().String("to", "", "to date")
	c.Flags().IntP("last", "l", 0, "last n periods")
	c.Flags().BoolP("diff", "d", false, "diff")
	c.Flags().BoolP("show-commodities", "s", false, "Show commodities on their own rows")
	c.Flags().Bool("days", false, "days")
	c.Flags().Bool("weeks", false, "weeks")
	c.Flags().Bool("months", false, "months")
	c.Flags().Bool("quarters", false, "quarters")
	c.Flags().Bool("years", false, "years")
	c.Flags().StringP("val", "v", "", "valuate in the given commodity")
	c.Flags().String("account", "", "filter accounts with a regex")
	c.Flags().String("commodity", "", "filter commodities with a regex")
	return c
}

func run(cmd *cobra.Command, args []string) {
	if err := execute(cmd, args); err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), err)
		os.Exit(1)
	}
}

func execute(cmd *cobra.Command, args []string) error {
	prof, err := cmd.Flags().GetString("cpuprofile")
	if err != nil {
		return err
	}
	if prof != "" {
		f, err := os.Create(prof)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	pipeline, err := configurePipeline(cmd, args)
	if err != nil {
		return err
	}
	var out = bufio.NewWriter(cmd.OutOrStdout())
	defer out.Flush()
	return processPipeline(out, pipeline)
}

type pipeline struct {
	Parser          parser.RecursiveParser
	Filter          ledger.Filter
	ProcessingSteps []ledger.Processor
	PerfCalc        performance.Calculator
}

func configurePipeline(cmd *cobra.Command, args []string) (*pipeline, error) {
	var (
		ctx = ledger.NewContext()
		err error
	)
	valuation, err := flags.GetCommodityFlag(cmd, ctx, "val")
	if err != nil {
		return nil, err
	}
	filterAccounts, err := flags.GetRegexFlag(cmd, "account")
	if err != nil {
		return nil, err
	}
	filterCommodities, err := flags.GetRegexFlag(cmd, "commodity")
	if err != nil {
		return nil, err
	}
	var (
		p = parser.RecursiveParser{
			File:    args[0],
			Context: ctx,
		}
		filter = ledger.Filter{
			CommoditiesFilter: filterCommodities,
			AccountsFilter:    filterAccounts,
		}
		bal   = balance.New(ctx, valuation)
		res   = new(performance.DailyPerfValues)
		steps = []ledger.Processor{
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
	)
	return &pipeline{
		Parser:          p,
		Filter:          filter,
		ProcessingSteps: steps,
		PerfCalc: performance.Calculator{
			Filter:    filter,
			Valuation: valuation,
		},
	}, nil
}

func processPipeline(w io.Writer, ppl *pipeline) error {
	var (
		l   ledger.Ledger
		err error
	)
	if l, err = ppl.Parser.BuildLedger(ledger.Filter{}); err != nil {
		return err
	}
	if err = l.Process(ppl.ProcessingSteps); err != nil {
		return err
	}
	for range ppl.PerfCalc.Perf(l) {
	}
	return nil
}
