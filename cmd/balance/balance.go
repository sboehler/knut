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
	"strconv"
	"strings"
	"time"

	"github.com/sboehler/knut/lib/balance"
	"github.com/sboehler/knut/lib/date"
	"github.com/sboehler/knut/lib/ledger"
	"github.com/sboehler/knut/lib/model/commodities"
	"github.com/sboehler/knut/lib/parser"
	"github.com/sboehler/knut/lib/report"

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

		RunE: run,
	}
	c.Flags().StringP("from", "", "", "from date")
	c.Flags().BoolP("diff", "d", false, "diff")
	c.Flags().BoolP("show-commodities", "s", false, "Show commodities on their own rows")
	c.Flags().StringP("to", "", "", "to date")
	c.Flags().BoolP("daily", "", false, "daily")
	c.Flags().BoolP("weekly", "", false, "weekly")
	c.Flags().BoolP("monthly", "", false, "monthly")
	c.Flags().BoolP("quarterly", "", false, "quarterly")
	c.Flags().BoolP("yearly", "", false, "yearly")
	c.Flags().StringArrayP("val", "v", []string{}, "valuate in the given commodity")
	c.Flags().StringArrayP("collapse", "c", []string{}, "<regex>,<level>")
	c.Flags().StringP("account", "", "", "filter accounts with a regex")
	c.Flags().StringP("commodity", "", "", "filter commodities with a regex")
	return c
}

func run(cmd *cobra.Command, args []string) error {
	o, err := parseOptions(cmd, args)
	if err != nil {
		return err
	}
	if err := createBalance(cmd, o); err != nil {
		return err
	}
	return nil
}

func init() {

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
	filterCommodities, err := cmd.Flags().GetString("commodity")
	if err != nil {
		return nil, err
	}

	return &options{
		File:              args[0],
		From:              from,
		To:                to,
		Valuations:        valuations,
		Diff:              diff,
		ShowCommodities:   showCommodities || len(valuations) == 0,
		FilterAccounts:    filterAccounts,
		FilterCommodities: filterCommodities,
		Period:            period,
		Collapse:          collapse,
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
	return &t, nil
}

func parsePeriod(cmd *cobra.Command, arg string) (*date.Period, error) {
	periods := []struct {
		name   string
		period date.Period
	}{
		{"daily", date.Daily},
		{"weekly", date.Weekly},
		{"monthly", date.Monthly},
		{"quarterly", date.Quarterly},
		{"yearly", date.Yearly}}
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
		regex := ""
		if len(s) == 2 {
			regex = s[1]
		}
		res = append(res, report.Collapse{Level: l, Regex: regex})
	}
	return res, nil
}

type options struct {
	File              string
	From              *time.Time
	To                *time.Time
	ShowCommodities   bool
	Diff              bool
	Period            *date.Period
	Valuations        []*commodities.Commodity
	FilterAccounts    string
	FilterCommodities string
	Collapse          []report.Collapse
}

func createLedgerOptions(o *options) ledger.Options {
	return ledger.Options{
		AccountsFilter:    o.FilterAccounts,
		CommoditiesFilter: o.FilterCommodities,
	}
}

func createDateSeries(o *options, l ledger.Ledger) []time.Time {
	if o.From == nil {
		if d, ok := l.MinDate(); ok {
			o.From = &d
		} else {
			return nil
		}
	}
	if o.To == nil {
		if d, ok := l.MaxDate(); ok {
			o.To = &d
		} else {
			return nil
		}
	}
	if o.Period != nil {
		return date.Series(*o.From, *o.To, *o.Period)
	}
	return []time.Time{*o.From, *o.To}
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
	lb := ledger.NewBuilder(createLedgerOptions(opts))
	if err := lb.Process(ch); err != nil {
		return err
	}
	l := lb.Build()
	b, err := process(opts.Valuations, createDateSeries(opts, l), l)
	if err != nil {
		return err
	}
	if opts.Diff {
		b = balance.Diffs(b)
	}
	r, err := report.NewReport(createReportOptions(opts), b)
	if err != nil {
		return err
	}
	out := bufio.NewWriter(cmd.OutOrStdout())
	defer out.Flush()
	report.NewRenderer(opts.ShowCommodities).Render(r).Render(out)
	return nil
}

// Options describes options for processing a ledger.
type Options struct {
	Commodities []*commodities.Commodity
	Dates       []time.Time
}

// process processes the ledger and creates valuations for the given commodities
// and returning balances for the given dates.
func process(commodities []*commodities.Commodity, dates []time.Time, l ledger.Ledger) ([]*balance.Balance, error) {
	balances := make([]*balance.Balance, 0, len(dates))
	balance := balance.New(commodities)
	step := 0
	for _, date := range dates {
		for step < len(l) && (l[step].Date == date || l[step].Date.Before(date)) {
			if err := balance.Update(l[step]); err != nil {
				return nil, err
			}
			step++
		}
		cur := balance.Copy()
		cur.Date = date
		balances = append(balances, cur)
	}
	return balances, nil
}
