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
	"io"
	"strings"
	"time"

	"github.com/sboehler/knut/lib/common/compare"
	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/common/date"
	"github.com/sboehler/knut/lib/common/dict"
	"github.com/sboehler/knut/lib/journal/printer"
	"github.com/sboehler/knut/lib/model"
	"github.com/sboehler/knut/lib/model/price"
	"github.com/sboehler/knut/lib/syntax"
	"github.com/sourcegraph/conc/pool"
)

// Builder represents an unprocessed
type Builder struct {
	days     map[time.Time]*Day
	min, max time.Time
}

// New creates a new Journal.
func New() *Builder {
	return &Builder{
		days: make(map[time.Time]*Day),
		min:  date.Date(9999, 12, 31),
		max:  time.Time{},
	}
}

// Day returns the Day for the given date.
func (j *Builder) Day(d time.Time) *Day {
	return dict.GetDefault(j.days, d, func() *Day { return &Day{Date: d} })
}

func (j *Builder) Build() *Journal {
	return &Journal{
		Days: dict.SortedValues(j.days, CompareDays),
	}
}

func (j *Builder) Add(d model.Directive) error {
	switch t := d.(type) {

	case *model.Price:
		d := j.Day(t.Date)
		if j.max.Before(d.Date) {
			j.max = d.Date
		}
		d.Prices = append(d.Prices, t)

	case *model.Open:
		d := j.Day(t.Date)
		d.Openings = append(d.Openings, t)

	case *model.Transaction:
		d := j.Day(t.Date)
		if j.max.Before(d.Date) {
			j.max = d.Date
		}
		if j.min.After(t.Date) {
			j.min = d.Date
		}
		d.Transactions = append(d.Transactions, t)

	case *model.Assertion:
		d := j.Day(t.Date)
		d.Assertions = append(d.Assertions, t)

	case *model.Close:
		d := j.Day(t.Date)
		d.Closings = append(d.Closings, t)

	default:
		return fmt.Errorf("unknown: %v (%T)", t, t)
	}
	return nil
}

func (j *Builder) Period() date.Period {
	return date.Period{Start: j.min, End: j.max}
}

func (j *Builder) Days(dates []time.Time) []*Day {
	var res []*Day
	for _, d := range dates {
		res = append(res, j.Day(d))
	}
	return res
}

func FromPath(ctx context.Context, reg *model.Registry, path string) (*Builder, error) {
	syntaxCh, worker1 := syntax.ParseFileRecursively(path)
	modelCh, worker2 := model.FromStream(reg, syntaxCh)
	journalCh, worker3 := FromModelStream(modelCh)
	p := pool.New().WithErrors().WithFirstError().WithContext(ctx)
	p.Go(worker1)
	p.Go(worker2)
	p.Go(worker3)
	if err := p.Wait(); err != nil {
		return nil, err
	}
	return <-journalCh, nil
}

func FromModelStream(modelCh <-chan []model.Directive) (<-chan *Builder, func(context.Context) error) {
	return cpr.FanIn(func(ctx context.Context, ch chan<- *Builder) error {
		j := New()
		err := cpr.ForEach(ctx, modelCh, func(directives []model.Directive) error {
			for _, d := range directives {
				if err := j.Add(d); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
		return cpr.Push(ctx, ch, j)
	})
}

type Journal struct {
	Days []*Day
}

func (j *Journal) Process(ps ...*Processor) error {
	var fs []func(*Day) error
	for _, proc := range ps {
		if proc != nil {
			fs = append(fs, proc.Process)
		}
	}
	_, err := cpr.Seq(context.Background(), j.Days, fs...)
	return err
}

// Day groups all commands for a given date.
type Day struct {
	Date         time.Time
	Prices       []*model.Price
	Assertions   []*model.Assertion
	Openings     []*model.Open
	Transactions []*model.Transaction
	Closings     []*model.Close

	Normalized price.NormalizedPrices

	Performance *Performance
}

// Less establishes an ordering on Day.
func CompareDays(d *Day, d2 *Day) compare.Order {
	return compare.Time(d.Date, d2.Date)
}

// Performance holds aggregate information used to compute
// portfolio performance.
type Performance struct {
	V0, V1, Inflow, Outflow, InternalInflow, InternalOutflow map[*model.Commodity]float64
	PortfolioInflow, PortfolioOutflow                        float64
}

func (p Performance) String() string {
	var buf strings.Builder
	for c, v := range p.V0 {
		fmt.Fprintf(&buf, "V0: %20s %f\n", c, v)
	}
	for c, f := range p.Inflow {
		fmt.Fprintf(&buf, "Inflow: %20s %f\n", c, f)
	}
	for c, f := range p.Outflow {
		fmt.Fprintf(&buf, "Outflow: %20s %f\n", c, f)
	}
	for c, f := range p.InternalInflow {
		fmt.Fprintf(&buf, "InternalInflow: %20s %f\n", c, f)
	}
	for c, f := range p.InternalOutflow {
		fmt.Fprintf(&buf, "InternalOutflow: %20s %f\n", c, f)
	}
	for c, v := range p.V1 {
		fmt.Fprintf(&buf, "V1: %20s %f\n", c, v)
	}
	return buf.String()
}

// PrintJournal prints a journal.
func Print(w io.Writer, j *Journal) error {
	p := printer.New(w)
	paddingUpdater := &Processor{
		Transaction: func(t *model.Transaction) error {
			p.UpdatePadding(t)
			return nil
		},
	}
	err := j.Process(
		Sort(),
		paddingUpdater,
	)
	if err != nil {
		return err
	}
	for _, day := range j.Days {
		for _, pr := range day.Prices {
			if _, err := p.PrintDirectiveLn(pr); err != nil {
				return err
			}
		}
		if len(day.Prices) > 0 {
			if _, err := io.WriteString(p, "\n"); err != nil {
				return err
			}
		}
		for _, o := range day.Openings {
			if _, err := p.PrintDirectiveLn(o); err != nil {
				return err
			}
		}
		if len(day.Openings) > 0 {
			if _, err := io.WriteString(p, "\n"); err != nil {
				return err
			}
		}
		for _, t := range day.Transactions {
			if _, err := p.PrintDirectiveLn(t); err != nil {
				return err
			}
		}
		for _, a := range day.Assertions {
			if _, err := p.PrintDirectiveLn(a); err != nil {
				return err
			}
		}
		if len(day.Assertions) > 0 {
			if _, err := io.WriteString(p, "\n"); err != nil {
				return err
			}
		}
		for _, c := range day.Closings {
			if _, err := p.PrintDirectiveLn(c); err != nil {
				return err
			}
		}
		if len(day.Closings) > 0 {
			if _, err := io.WriteString(p, "\n"); err != nil {
				return err
			}
		}
	}
	return nil
}

type Processor struct {
	DayStart    func(*Day) error
	Price       func(*model.Price) error
	Open        func(*model.Open) error
	Transaction func(*model.Transaction) error
	Posting     func(*model.Transaction, *model.Posting) error
	Assertion   func(*model.Assertion) error
	Balance     func(*model.Assertion, *model.Balance) error
	Close       func(*model.Close) error
	DayEnd      func(*Day) error
}

func (proc *Processor) Process(d *Day) error {
	if proc.DayStart != nil {
		if err := proc.DayStart(d); err != nil {
			return err
		}
	}
	if proc.Price != nil {
		for _, p := range d.Prices {
			if err := proc.Price(p); err != nil {
				return err
			}
		}
	}
	if proc.Open != nil {
		for _, o := range d.Openings {
			if err := proc.Open(o); err != nil {
				return err
			}
		}
	}
	if proc.Transaction != nil {
		for _, t := range d.Transactions {
			if err := proc.Transaction(t); err != nil {
				return err
			}
			if proc.Posting != nil {
				for _, p := range t.Postings {
					if err := proc.Posting(t, p); err != nil {
						return err
					}
				}
			}
		}
	} else if proc.Posting != nil {
		for _, t := range d.Transactions {
			for _, p := range t.Postings {
				if err := proc.Posting(t, p); err != nil {
					return err
				}
			}
		}
	}
	if proc.Assertion != nil {
		for _, a := range d.Assertions {
			if err := proc.Assertion(a); err != nil {
				return err
			}
			if proc.Balance != nil {
				for i := range a.Balances {
					if err := proc.Balance(a, &a.Balances[i]); err != nil {
						return err
					}
				}
			}
		}
	} else if proc.Balance != nil {
		for _, a := range d.Assertions {
			for i := range a.Balances {
				if err := proc.Balance(a, &a.Balances[i]); err != nil {
					return err
				}
			}
		}
	}
	if proc.Close != nil {
		for _, a := range d.Closings {
			if err := proc.Close(a); err != nil {
				return err
			}
		}
	}
	if proc.DayEnd != nil {
		if err := proc.DayEnd(d); err != nil {
			return err
		}
	}
	return nil
}
