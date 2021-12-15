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

package past

import (
	"fmt"
	"sort"
	"time"

	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
)

// FromDirectives reads directives from the given channel and
// builds a Ledger if successful.
func FromDirectives(ctx journal.Context, filter journal.Filter, results <-chan interface{}) (*ast.PAST, error) {
	var b = NewBuilder(ctx, filter)
	for res := range results {
		switch t := res.(type) {
		case error:
			return nil, t
		case *ast.Open:
			b.AddOpening(t)
		case *ast.Price:
			b.AddPrice(t)
		case *ast.Transaction:
			b.AddTransaction(t)
		case *ast.Assertion:
			b.AddAssertion(t)
		case *ast.Value:
			b.AddValue(t)
		case *ast.Close:
			b.AddClosing(t)
		default:
			return nil, fmt.Errorf("unknown: %#v", t)
		}
	}
	return b.Build(), nil
}

// Builder maps dates to days
type Builder struct {
	filter  journal.Filter
	days    map[time.Time]*ast.Day
	Context journal.Context
}

// NewBuilder creates a new builder.
func NewBuilder(ctx journal.Context, f journal.Filter) *Builder {
	return &Builder{f, make(map[time.Time]*ast.Day), ctx}
}

// Build creates a new
func (b *Builder) Build() *ast.PAST {
	var result = make([]*ast.Day, 0, len(b.days))
	for _, s := range b.days {
		result = append(result, s)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Date.Before(result[j].Date)
	})
	return &ast.PAST{
		Days:    result,
		Context: b.Context,
	}

}

func (b *Builder) getOrCreate(d time.Time) *ast.Day {
	s, ok := b.days[d]
	if !ok {
		s = &ast.Day{Date: d}
		b.days[d] = s
	}
	return s
}

// AddTransaction adds a transaction directive.
func (b *Builder) AddTransaction(t *ast.Transaction) {
	if len(t.AddOns) > 0 {
		for _, addOn := range t.AddOns {
			switch a := addOn.(type) {
			case *ast.Accrual:
				for _, ts := range a.Expand(t) {
					b.AddTransaction(ts)
				}
			}
		}
		return
	}
	var filtered []ast.Posting
	for _, p := range t.Postings {
		if p.Matches(b.filter) {
			filtered = append(filtered, p)
		}
	}
	if len(filtered) > 0 {
		t.Postings = filtered
		var s = b.getOrCreate(t.Date)
		s.Transactions = append(s.Transactions, t)
	}
}

// AddOpening adds an open directive.
func (b *Builder) AddOpening(o *ast.Open) {
	var s = b.getOrCreate(o.Date)
	s.Openings = append(s.Openings, o)
}

// AddClosing adds a close directive.
func (b *Builder) AddClosing(close *ast.Close) {
	if b.filter.MatchAccount(close.Account) {
		var s = b.getOrCreate(close.Date)
		s.Closings = append(s.Closings, close)
	}
}

// AddPrice adds a price directive.
func (b *Builder) AddPrice(p *ast.Price) {
	var s = b.getOrCreate(p.Date)
	s.Prices = append(s.Prices, p)
}

// AddAssertion adds an assertion directive.
func (b *Builder) AddAssertion(a *ast.Assertion) {
	if b.filter.MatchAccount(a.Account) && b.filter.MatchCommodity(a.Commodity) {
		var s = b.getOrCreate(a.Date)
		s.Assertions = append(s.Assertions, a)
	}
}

// AddValue adds an value directive.
func (b *Builder) AddValue(a *ast.Value) {
	if b.filter.MatchAccount(a.Account) && b.filter.MatchCommodity(a.Commodity) {
		var s = b.getOrCreate(a.Date)
		s.Values = append(s.Values, a)
	}
}
