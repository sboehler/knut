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

package journal

import (
	"context"
	"fmt"
	"time"

	"github.com/sboehler/knut/lib/common/compare"
	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/common/date"
	"github.com/sboehler/knut/lib/common/dict"
	"github.com/sboehler/knut/lib/common/slice"
	"go.uber.org/multierr"
)

// Journal represents an unprocessed
type Journal struct {
	Context  Context
	Days     map[time.Time]*Day
	min, max time.Time
}

// New creates a new Journal.
func New(ctx Context) *Journal {
	return &Journal{
		Context: ctx,
		Days:    make(map[time.Time]*Day),
		min:     date.Date(9999, 12, 31),
		max:     time.Time{},
	}
}

// day returns the day for the given date.
func (ast *Journal) day(d time.Time) *Day {
	return dict.GetDefault(ast.Days, d, func() *Day { return &Day{Date: d} })
}

// SortedDays returns all days ordered by date.
func (ast *Journal) SortedDays() []*Day {
	var res []*Day
	for _, day := range ast.Days {
		compare.Sort(day.Transactions, CompareTransactions)
		res = append(res, day)
	}
	compare.Sort(res, CompareDays)
	return res
}

// AddOpen adds an Open directive.
func (ast *Journal) AddOpen(o *Open) {
	d := ast.day(o.Date)
	d.Openings = append(d.Openings, o)
}

// AddPrice adds an Price directive.
func (ast *Journal) AddPrice(p *Price) {
	d := ast.day(p.Date)
	d.Prices = append(d.Prices, p)
}

// AddTransaction adds an Transaction directive.
func (ast *Journal) AddTransaction(t *Transaction) {
	d := ast.day(t.Date)
	if ast.max.Before(d.Date) {
		ast.max = d.Date
	}
	if ast.min.After(t.Date) {
		ast.min = d.Date
	}
	d.Transactions = append(d.Transactions, t)
}

// AddValue adds an Value directive.
func (ast *Journal) AddValue(v *Value) {
	d := ast.day(v.Date)
	d.Values = append(d.Values, v)
}

// AddAssertion adds an Assertion directive.
func (ast *Journal) AddAssertion(a *Assertion) {
	d := ast.day(a.Date)
	d.Assertions = append(d.Assertions, a)
}

// AddClose adds an Close directive.
func (ast *Journal) AddClose(c *Close) {
	d := ast.day(c.Date)
	d.Closings = append(d.Closings, c)
}

func (ast *Journal) Min() time.Time {
	return ast.min
}

func (ast *Journal) Max() time.Time {
	return ast.max
}

func (ast *Journal) Process(fs ...func(*Day) error) (*Ledger, error) {
	ds := dict.SortedValues(ast.Days, CompareDays)
	err := slice.Parallel(context.Background(), ds, fs...)
	if err != nil {
		return nil, err
	}
	return &Ledger{
		Context: ast.Context,
		Days:    ds,
	}, nil

}

func FromPath(ctx context.Context, jctx Context, path string) (*Journal, error) {
	builder := New(jctx)
	p := RecursiveParser{
		Context: jctx,
		File:    path,
	}
	var errs error
	err := cpr.Consume(ctx, p.Parse(ctx), func(d any) error {
		switch t := d.(type) {

		case error:
			errs = multierr.Append(errs, t)

		case *Open:
			builder.AddOpen(t)

		case *Price:
			builder.AddPrice(t)

		case *Transaction:
			if t.Accrual != nil {
				for _, ts := range t.Accrual.Expand(t) {
					builder.AddTransaction(ts)
				}
			} else {
				builder.AddTransaction(t)
			}

		case *Assertion:
			builder.AddAssertion(t)

		case *Value:
			builder.AddValue(t)

		case *Close:
			builder.AddClose(t)

		default:
			errs = multierr.Append(errs, fmt.Errorf("unknown: %#v", t))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if errs != nil {
		return nil, errs
	}
	return builder, nil
}

// Ledger is an ordered and processed list of Days.
type Ledger struct {
	Context Context
	Days    []*Day
}

// Day groups all commands for a given date.
type Day struct {
	Date         time.Time
	Prices       []*Price
	Assertions   []*Assertion
	Values       []*Value
	Openings     []*Open
	Transactions []*Transaction
	Closings     []*Close

	Amounts, Value Amounts

	Normalized NormalizedPrices

	Performance *Performance
}

// Less establishes an ordering on Day.
func CompareDays(d *Day, d2 *Day) compare.Order {
	return compare.Time(d.Date, d2.Date)
}

// Performance holds aggregate information used to compute
// portfolio performance.
type Performance struct {
	V0, V1, Inflow, Outflow, InternalInflow, InternalOutflow map[*Commodity]float64
	PortfolioInflow, PortfolioOutflow                        float64
}
