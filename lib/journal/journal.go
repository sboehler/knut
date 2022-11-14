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

// Day returns the Day for the given date.
func (j *Journal) Day(d time.Time) *Day {
	return dict.GetDefault(j.Days, d, func() *Day { return &Day{Date: d} })
}

func (j *Journal) SortedDays() *Ledger {
	ds := dict.SortedValues(j.Days, CompareDays)
	for _, day := range ds {
		compare.Sort(day.Transactions, CompareTransactions)
	}
	return &Ledger{
		Context: j.Context,
		Days:    ds,
	}
}

// AddOpen adds an Open directive.
func (j *Journal) AddOpen(o *Open) {
	d := j.Day(o.Date)
	d.Openings = append(d.Openings, o)
}

// AddPrice adds an Price directive.
func (j *Journal) AddPrice(p *Price) {
	d := j.Day(p.Date)
	d.Prices = append(d.Prices, p)
}

// AddTransaction adds an Transaction directive.
func (j *Journal) AddTransaction(t *Transaction) {
	d := j.Day(t.Date)
	if j.max.Before(d.Date) {
		j.max = d.Date
	}
	if j.min.After(t.Date) {
		j.min = d.Date
	}
	d.Transactions = append(d.Transactions, t)
}

// AddValue adds an Value directive.
func (j *Journal) AddValue(v *Value) {
	d := j.Day(v.Date)
	d.Values = append(d.Values, v)
}

// AddAssertion adds an Assertion directive.
func (j *Journal) AddAssertion(a *Assertion) {
	d := j.Day(a.Date)
	d.Assertions = append(d.Assertions, a)
}

// AddClose adds an Close directive.
func (j *Journal) AddClose(c *Close) {
	d := j.Day(c.Date)
	d.Closings = append(d.Closings, c)
}

func (j *Journal) Min() time.Time {
	return j.min
}

func (j *Journal) Max() time.Time {
	return j.max
}

func (j *Journal) Process(fs ...func(*Day) error) (*Ledger, error) {
	ds := dict.SortedValues(j.Days, CompareDays)
	ds, err := slice.Parallel(ds, fs...)
	if err != nil {
		return nil, err
	}
	return &Ledger{
		Context: j.Context,
		Days:    ds,
	}, nil
}

func FromPath(ctx context.Context, jctx Context, path string) (*Journal, error) {
	j := New(jctx)
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
			j.AddOpen(t)

		case *Price:
			j.AddPrice(t)

		case *Transaction:
			if t.Accrual != nil {
				for _, ts := range t.Accrual.Expand(t) {
					j.AddTransaction(ts)
				}
			} else {
				j.AddTransaction(t)
			}

		case *Assertion:
			j.AddAssertion(t)

		case *Value:
			j.AddValue(t)

		case *Close:
			j.AddClose(t)

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
	return j, nil
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

	Value Amounts

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
