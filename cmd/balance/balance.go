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

package balance

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"runtime/pprof"
	"strconv"
	"strings"
	"time"

	"github.com/sboehler/knut/cmd/flags"
	"github.com/sboehler/knut/lib/balance"
	"github.com/sboehler/knut/lib/date"
	"github.com/sboehler/knut/lib/ledger"
	"github.com/sboehler/knut/lib/model/accounts"
	"github.com/sboehler/knut/lib/model/commodities"
	"github.com/sboehler/knut/lib/parser"
	"github.com/sboehler/knut/lib/report"
	"github.com/sboehler/knut/lib/table"

	"github.com/spf13/cobra"
	"go.uber.org/multierr"
)

// CreateCmd creates the command.
func CreateCmd() *cobra.Command {

	// Cmd is the balance command.
	var c = &cobra.Command{
		Use:   "balance",
		Short: "create a balance sheet",
		Long:  `Compute a balance for a date or set of dates.`,

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
	c.Flags().StringArrayP("collapse", "c", nil, "<level>,<regex>")
	c.Flags().String("account", "", "filter accounts with a regex")
	c.Flags().String("commodity", "", "filter commodities with a regex")
	c.Flags().Bool("close", false, "close income and expenses accounts after every period")
	c.Flags().Int32("digits", 0, "round to number of digits")
	c.Flags().BoolP("thousands", "k", false, "show numbers in units of 1000")
	c.Flags().Bool("color", false, "print output in color")
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
	Accounts       *accounts.Accounts
	Parser         parser.RecursiveParser
	Filter         ledger.Filter
	BalanceBuilder balance.Builder
	ReportBuilder  report.Builder
	ReportRenderer report.Renderer
	TextRenderer   table.TextRenderer
}

func configurePipeline(cmd *cobra.Command, args []string) (*pipeline, error) {
	var (
		from, to *time.Time
		err      error
	)
	if cmd.Flags().Changed("from") {
		if from, err = flags.GetDateFlag(cmd, "from"); err != nil {
			return nil, err
		}
	}
	if cmd.Flags().Changed("to") {
		if to, err = flags.GetDateFlag(cmd, "to"); err != nil {
			return nil, err
		}
	} else {
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
	var valuation *commodities.Commodity
	if cmd.Flags().Changed("val") {
		if valuation, err = flags.GetCommodityFlag(cmd, "val"); err != nil {
			return nil, err
		}
	}
	showCommodities, err := cmd.Flags().GetBool("show-commodities")
	if err != nil {
		return nil, err
	}
	diff, err := cmd.Flags().GetBool("diff")
	if err != nil {
		return nil, err
	}
	digits, err := cmd.Flags().GetInt32("digits")
	if err != nil {
		return nil, err
	}
	close, err := cmd.Flags().GetBool("close")
	if err != nil {
		return nil, err
	}
	period, err := parsePeriod(cmd, "period")
	if err != nil {
		return nil, err
	}
	collapse, err := parseCollapse(cmd, "collapse")
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
	thousands, err := cmd.Flags().GetBool("thousands")
	if err != nil {
		return nil, err
	}
	color, err := cmd.Flags().GetBool("color")
	if err != nil {
		return nil, err
	}

	var (
		parser = parser.RecursiveParser{
			File:     args[0],
			Accounts: accounts.New(),
		}
		balanceBuilder = balance.Builder{
			From:      from,
			To:        to,
			Period:    period,
			Last:      last,
			Valuation: valuation,
			Close:     close,
			Diff:      diff,
		}
		filter = ledger.Filter{
			AccountsFilter:    filterAccounts,
			CommoditiesFilter: filterCommodities,
		}
		reportBuilder = report.Builder{
			Value:    valuation != nil,
			Collapse: collapse,
		}
		reportRenderer = report.Renderer{
			Commodities: showCommodities || valuation == nil,
		}
		tableRenderer = table.TextRenderer{
			Color:     color,
			Thousands: thousands,
			Round:     digits,
		}
	)
	return &pipeline{
		Parser:         parser,
		Filter:         filter,
		BalanceBuilder: balanceBuilder,
		ReportBuilder:  reportBuilder,
		ReportRenderer: reportRenderer,
		TextRenderer:   tableRenderer,
	}, nil
}

func processPipeline(w io.Writer, ppl *pipeline) error {
	var (
		l   ledger.Ledger
		bal []*balance.Balance
		r   *report.Report
		err error
	)
	if l, err = ppl.Parser.BuildLedger(ppl.Filter); err != nil {
		return err
	}
	if bal, err = ppl.BalanceBuilder.Build(l); err != nil {
		return err
	}
	if r, err = ppl.ReportBuilder.Build(bal); err != nil {
		return err
	}
	return ppl.TextRenderer.Render(ppl.ReportRenderer.Render(r), w)
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

func parseCollapse(cmd *cobra.Command, name string) ([]report.Collapse, error) {
	collapse, err := cmd.Flags().GetStringArray(name)
	if err != nil {
		return nil, err
	}
	var res = make([]report.Collapse, 0, len(collapse))
	for _, c := range collapse {
		var s = strings.SplitN(c, ",", 2)
		l, err := strconv.Atoi(s[0])
		if err != nil {
			return nil, fmt.Errorf("expected integer level, got %q (error: %v)", s[0], err)
		}
		var regex *regexp.Regexp
		if len(s) == 2 {
			if regex, err = regexp.Compile(s[1]); err != nil {
				return nil, err
			}
		}
		res = append(res, report.Collapse{Level: l, Regex: regex})
	}
	return res, nil
}
