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
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/sboehler/knut/lib/balance"
	"github.com/sboehler/knut/lib/date"
	"github.com/sboehler/knut/lib/ledger"
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
	c := &cobra.Command{
		Use:   "balance",
		Short: "create a balance sheet",
		Long:  `Compute a balance for a date or set of dates.`,

		Args: cobra.ExactValidArgs(1),

		Run: run,
	}
	c.Flags().StringP("from", "", "", "from date")
	c.Flags().StringP("to", "", "", "to date")
	c.Flags().IntP("last", "l", 0, "last n periods")
	c.Flags().BoolP("diff", "d", false, "diff")
	c.Flags().BoolP("show-commodities", "s", false, "Show commodities on their own rows")
	c.Flags().BoolP("days", "", false, "days")
	c.Flags().BoolP("weeks", "", false, "weeks")
	c.Flags().BoolP("months", "", false, "months")
	c.Flags().BoolP("quarters", "", false, "quarters")
	c.Flags().BoolP("years", "", false, "years")
	c.Flags().StringArrayP("val", "v", []string{}, "valuate in the given commodity")
	c.Flags().StringArrayP("collapse", "c", []string{}, "<level>,<regex>")
	c.Flags().StringP("account", "", "", "filter accounts with a regex")
	c.Flags().StringP("commodity", "", "", "filter commodities with a regex")
	c.Flags().BoolP("close", "", false, "close income and expenses accounts after every period")
	c.Flags().Int32P("digits", "", 0, "round to number of digits")
	c.Flags().BoolP("thousands", "k", false, "show numbers in units of 1000")
	c.Flags().BoolP("color", "", false, "print output in color")
	return c
}

func run(cmd *cobra.Command, args []string) {
	if err := execute(cmd, args); err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), err)
		os.Exit(1)
	}
}

func execute(cmd *cobra.Command, args []string) error {
	o, err := parseOptions(cmd, args)
	if err != nil {
		return err
	}
	if err := createBalance(cmd, o); err != nil {
		return err
	}
	return nil
}

func parseOptions(cmd *cobra.Command, args []string) (*options, error) {
	from, err := parseDate(cmd, "from")
	if err != nil {
		return nil, err
	}
	to, err := parseDate(cmd, "to")
	if err != nil {
		return nil, err
	}
	last, err := cmd.Flags().GetInt("last")
	if err != nil {
		return nil, err
	}
	valuations, err := parseValuations(cmd, "val")
	if err != nil {
		return nil, err
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
	thousands, err := cmd.Flags().GetBool("thousands")
	if err != nil {
		return nil, err
	}
	color, err := cmd.Flags().GetBool("color")
	if err != nil {
		return nil, err
	}

	return &options{
		File:              args[0],
		From:              from,
		To:                to,
		Last:              last,
		Valuations:        valuations,
		Diff:              diff,
		ShowCommodities:   showCommodities || len(valuations) == 0,
		FilterAccounts:    filterAccountsRegex,
		FilterCommodities: filterCommoditiesRegex,
		Period:            period,
		Collapse:          collapse,
		Close:             close,
		RoundDigits:       digits,
		Thousands:         thousands,
		Color:             color,
	}, nil
}

func parseValuations(cmd *cobra.Command, name string) ([]*commodities.Commodity, error) {
	vals, err := cmd.Flags().GetStringArray(name)
	if err != nil {
		return nil, err
	}
	var valuations = make([]*commodities.Commodity, len(vals))
	for i, v := range vals {
		valuations[i] = commodities.Get(v)
	}
	return valuations, nil
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
	periods := []struct {
		name   string
		period date.Period
	}{
		{"days", date.Daily},
		{"weeks", date.Weekly},
		{"months", date.Monthly},
		{"quarters", date.Quarterly},
		{"years", date.Yearly}}
	var (
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
			r := tuple.period
			result = &r
		}
	}
	return result, errors
}

var defaultRegex = regexp.MustCompile("")

func parseCollapse(cmd *cobra.Command, name string) ([]report.Collapse, error) {
	collapse, err := cmd.Flags().GetStringArray(name)
	if err != nil {
		return nil, err
	}
	res := make([]report.Collapse, 0, len(collapse))
	for _, c := range collapse {
		s := strings.SplitN(c, ",", 2)
		l, err := strconv.Atoi(s[0])
		if err != nil {
			return nil, fmt.Errorf("Expected integer level, got %q (error: %v)", s[0], err)
		}
		regex := defaultRegex
		if len(s) == 2 {
			if regex, err = regexp.Compile(s[1]); err != nil {
				return nil, err
			}
		}
		res = append(res, report.Collapse{Level: l, Regex: regex})
	}
	return res, nil
}

type options struct {
	File                                           string
	From, To                                       *time.Time
	Last                                           int
	Period                                         *date.Period
	Valuations                                     []*commodities.Commodity
	FilterAccounts, FilterCommodities              *regexp.Regexp
	Collapse                                       []report.Collapse
	RoundDigits                                    int32
	ShowCommodities, Diff, Close, Thousands, Color bool
}

func createLedgerOptions(o *options) ledger.Options {
	return ledger.Options{
		AccountsFilter:    o.FilterAccounts,
		CommoditiesFilter: o.FilterCommodities,
	}
}

func createDateSeries(o *options, l ledger.Ledger) []time.Time {
	var from, to time.Time
	if o.From != nil {
		from = *o.From
	} else if d, ok := l.MinDate(); ok {
		from = d
	} else {
		return nil
	}
	if o.To != nil {
		to = *o.To
	} else if d, ok := l.MaxDate(); ok {
		to = d
	} else {
		return nil
	}
	if o.Period != nil {
		return date.Series(from, to, *o.Period)
	}
	return []time.Time{from, to}
}

func createReportOptions(o *options) report.Options {
	var val *int
	if len(o.Valuations) > 0 {
		v := 0
		val = &v
	}
	return report.Options{
		Valuation: val,
		Collapse:  o.Collapse,
	}
}

func createBalance(cmd *cobra.Command, opts *options) error {
	ch, err := parser.Parse(opts.File)
	if err != nil {
		return err
	}
	l, err := ledger.Build(createLedgerOptions(opts), ch)
	if err != nil {
		return err
	}
	b, err := process(opts, l)
	if err != nil {
		return err
	}
	r, err := report.NewReport(createReportOptions(opts), b)
	if err != nil {
		return err
	}
	out := bufio.NewWriter(cmd.OutOrStdout())
	defer out.Flush()

	tb := report.NewRenderer(report.Config{Commodities: opts.ShowCommodities}).Render(r)

	return table.NewConsoleRenderer(tb, opts.Color, opts.Thousands, opts.RoundDigits).Render(out)
}

// process processes the ledger and creates valuations for the given commodities
// and returning balances for the given dates.
func process(opts *options, l ledger.Ledger) ([]*balance.Balance, error) {
	dates := createDateSeries(opts, l)
	balances := make([]*balance.Balance, 0, len(dates))
	bal := balance.New(opts.Valuations)
	day := 0
	for _, date := range dates {
		for day < len(l) && (l[day].Date == date || l[day].Date.Before(date)) {
			if err := bal.Update(l[day]); err != nil {
				return nil, err
			}
			day++
		}
		cur := bal.Copy()
		cur.Date = date
		balances = append(balances, cur)
		bal.CloseIncomeAndExpenses = opts.Close
	}
	if opts.Diff {
		balances = balance.Diffs(balances)
	}
	if opts.Last > 0 && opts.Last < len(balances) {
		balances = balances[len(balances)-opts.Last:]
	}
	return balances, nil
}
