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

package ledger

import (
	"fmt"
	"sort"
	"time"
)

// FromDirectives reads directives from the given channel and
// builds a Ledger if successful.
func FromDirectives(ctx Context, filter Filter, results <-chan interface{}) (Ledger, error) {
	var b = NewBuilder(ctx, filter)
	for res := range results {
		switch t := res.(type) {
		case error:
			return Ledger{}, t
		case Open:
			b.AddOpening(t)
		case *Price:
			b.AddPrice(t)
		case *Transaction:
			b.AddTransaction(t)
		case *Assertion:
			b.AddAssertion(t)
		case *Value:
			b.AddValue(t)
		case Close:
			b.AddClosing(t)
		case Accrual:
			b.AddAccrual(t)
		default:
			return Ledger{}, fmt.Errorf("unknown: %#v", t)
		}
	}
	return b.Build(), nil
}

// Builder maps dates to days
type Builder struct {
	filter  Filter
	days    map[time.Time]*Day
	Context Context
}

// NewBuilder creates a new builder.
func NewBuilder(ctx Context, f Filter) *Builder {
	return &Builder{f, make(map[time.Time]*Day), ctx}
}

// Build creates a new
func (b *Builder) Build() Ledger {
	var result = make([]*Day, 0, len(b.days))
	for _, s := range b.days {
		result = append(result, s)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Date.Before(result[j].Date)
	})
	return Ledger{
		Days:    result,
		Context: b.Context,
	}

}

func (b *Builder) getOrCreate(d time.Time) *Day {
	s, ok := b.days[d]
	if !ok {
		s = &Day{Date: d}
		b.days[d] = s
	}
	return s
}

// AddTransaction adds a transaction directive.
func (b *Builder) AddTransaction(t *Transaction) {
	var filtered []Posting
	for _, p := range t.Postings {
		if b.filter.MatchPosting(p) {
			filtered = append(filtered, p)
		}
	}
	if len(filtered) > 0 {
		t.Postings = filtered
		var s = b.getOrCreate(t.Date)
		s.Transactions = append(s.Transactions, t)
	}
}

// AddAccrual adds an accrual directive.
func (b *Builder) AddAccrual(t Accrual) {
	for _, t := range t.Expand() {
		b.AddTransaction(t)
	}
}

// AddOpening adds an open directive.
func (b *Builder) AddOpening(o Open) {
	var s = b.getOrCreate(o.Date)
	s.Openings = append(s.Openings, o)
}

// AddClosing adds a close directive.
func (b *Builder) AddClosing(close Close) {
	if b.filter.MatchAccount(close.Account) {
		var s = b.getOrCreate(close.Date)
		s.Closings = append(s.Closings, close)
	}
}

// AddPrice adds a price directive.
func (b *Builder) AddPrice(p *Price) {
	var s = b.getOrCreate(p.Date)
	s.Prices = append(s.Prices, p)
}

// AddAssertion adds an assertion directive.
func (b *Builder) AddAssertion(a *Assertion) {
	if b.filter.MatchAccount(a.Account) && b.filter.MatchCommodity(a.Commodity) {
		var s = b.getOrCreate(a.Date)
		s.Assertions = append(s.Assertions, a)
	}
}

// AddValue adds an value directive.
func (b *Builder) AddValue(a *Value) {
	if b.filter.MatchAccount(a.Account) && b.filter.MatchCommodity(a.Commodity) {
		var s = b.getOrCreate(a.Date)
		s.Values = append(s.Values, a)
	}
}
