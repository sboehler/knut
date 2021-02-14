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
	"regexp"
	"runtime/pprof"
	"time"

	"github.com/sboehler/knut/lib/balance"
	"github.com/sboehler/knut/lib/date"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/ledger"
	"github.com/sboehler/knut/lib/model/commodities"
	"github.com/sboehler/knut/lib/performance"

	"github.com/spf13/cobra"
	"go.uber.org/multierr"
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
	Journal        journal.Journal
	LedgerFilter   ledger.Filter
	BalanceBuilder balance.Builder
	PerfCalc       performance.Calculator
}

func configurePipeline(cmd *cobra.Command, args []string) (*pipeline, error) {
	var (
		from, to *time.Time
		err      error
	)
	if from, err = parseDate(cmd, "from"); err != nil {
		return nil, err
	}
	if to, err = parseDate(cmd, "to"); err != nil {
		return nil, err
	}
	if to == nil {
		var (
			now = time.Now()
			d   = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		)
		to = &d
	}
	last, err := cmd.Flags().GetInt("last")
	if err != nil {
		return nil, err
	}
	valuation, err := parseValuation(cmd, "val")
	if err != nil {
		return nil, err
	}
	// showCommodities, err := cmd.Flags().GetBool("show-commodities")
	// if err != nil {
	// 	return nil, err
	// }
	period, err := parsePeriod(cmd, "period")
	if err != nil {
		return nil, err
	}
	filterAccounts, err := cmd.Flags().GetString("account")
	if err != nil {
		return nil, err
	}
	filterAccountsRegex, err := regexp.Compile(filterAccounts)
	if err != nil {
		return nil, err
	}
	filterCommodities, err := cmd.Flags().GetString("commodity")
	if err != nil {
		return nil, err
	}
	filterCommoditiesRegex, err := regexp.Compile(filterCommodities)
	if err != nil {
		return nil, err
	}

	var (
		journal = journal.Journal{
			File: args[0],
		}
		balanceBuilder = balance.Builder{
			From:      from,
			To:        to,
			Period:    period,
			Last:      last,
			Valuation: valuation,
		}
		ledgerFilter = ledger.Filter{
			CommoditiesFilter: filterCommoditiesRegex,
			AccountsFilter:    filterAccountsRegex,
		}
	)
	return &pipeline{
		Journal:        journal,
		LedgerFilter:   ledgerFilter,
		BalanceBuilder: balanceBuilder,
		PerfCalc: performance.Calculator{
			Filter:    ledgerFilter,
			Valuation: valuation,
		},
	}, nil
}

func processPipeline(w io.Writer, ppl *pipeline) error {
	var (
		l   ledger.Ledger
		err error
	)
	if l, err = ledger.FromDirectives(ledger.Filter{}, ppl.Journal.Parse()); err != nil {
		return err
	}
	if _, err = ppl.BalanceBuilder.Build(l); err != nil {
		return err
	}
	for range ppl.PerfCalc.Perf(l) {
	}
	return nil
}

func parseValuation(cmd *cobra.Command, name string) (*commodities.Commodity, error) {
	val, err := cmd.Flags().GetString(name)
	if err != nil {
		return nil, err
	}
	if len(val) == 0 {
		return nil, nil
	}
	return commodities.Get(val), nil
}

func parseDate(cmd *cobra.Command, arg string) (*time.Time, error) {
	s, err := cmd.Flags().GetString(arg)
	if err != nil || s == "" {
		return nil, err
	}
	t, err := time.Parse("2006-01-02", s)
	return &t, err
}

func parsePeriod(cmd *cobra.Command, arg string) (*date.Period, error) {
	var (
		periods = []struct {
			name   string
			period date.Period
		}{
			{"days", date.Daily},
			{"weeks", date.Weekly},
			{"months", date.Monthly},
			{"quarters", date.Quarterly},
			{"years", date.Yearly},
		}

		errors error
		result *date.Period
	)
	for _, tuple := range periods {
		v, err := cmd.Flags().GetBool(tuple.name)
		if err != nil {
			errors = multierr.Append(errors, err)
			continue
		}
		if v && result == nil {
			var p = tuple.period
			result = &p
		}
	}
	return result, errors
}
