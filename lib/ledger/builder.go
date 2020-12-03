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

package ledger

import (
	"fmt"
	"regexp"
	"sort"
	"time"

	"github.com/sboehler/knut/lib/model"
)

// Builder maps dates to steps
type Builder struct {
	accountFilter, commodityFilter *regexp.Regexp
	steps                          map[time.Time]*Step
}

// Options represents configuration options for creating a ledger.
type Options struct {
	AccountsFilter, CommoditiesFilter *regexp.Regexp
}

// NewBuilder creates a new builder.
func NewBuilder(options Options) *Builder {
	var af, cf *regexp.Regexp
	if options.AccountsFilter != nil {
		af = options.AccountsFilter
	} else {
		af = regexp.MustCompile("")
	}
	if options.CommoditiesFilter != nil {
		cf = options.CommoditiesFilter
	} else {
		cf = regexp.MustCompile("")
	}
	return &Builder{
		accountFilter:   af,
		commodityFilter: cf,
		steps:           make(map[time.Time]*Step),
	}
}

// Process creates a new ledger from the results channel.
func (b *Builder) Process(results <-chan interface{}) error {
	for res := range results {
		switch t := res.(type) {
		case error:
			return t
		case *model.Open:
			b.AddOpening(t)
		case *model.Price:
			b.AddPrice(t)
		case *model.Transaction:
			b.AddTransaction(t)
		case *model.Assertion:
			b.AddAssertion(t)
		case *model.Value:
			b.AddValue(t)
		case *model.Close:
			b.AddClosing(t)
		default:
			return fmt.Errorf("Unknown: %v", t)
		}
	}
	return nil
}

// Build creates a new ledger.
func (b *Builder) Build() Ledger {
	var result = make([]*Step, 0, len(b.steps))
	for _, s := range b.steps {
		result = append(result, s)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Date.Before(result[j].Date)
	})
	return result

}

func (b *Builder) getOrCreate(d time.Time) *Step {
	s, ok := b.steps[d]
	if !ok {
		s = &Step{Date: d}
		b.steps[d] = s
	}
	return s
}

// AddTransaction adds a transaction directive.
func (b *Builder) AddTransaction(t *model.Transaction) {
	for _, p := range t.Postings {
		if (b.accountFilter.MatchString(p.Credit.String()) ||
			b.accountFilter.MatchString(p.Debit.String())) &&
			b.commodityFilter.MatchString(p.Commodity.String()) {
			s := b.getOrCreate(t.Date())
			s.Transactions = append(s.Transactions, t)
			return
		}
	}
}

// AddOpening adds an open directive.
func (b *Builder) AddOpening(o *model.Open) {
	s := b.getOrCreate(o.Date())
	s.Openings = append(s.Openings, o)
}

// AddClosing adds a close directive.
func (b *Builder) AddClosing(close *model.Close) {
	s := b.getOrCreate(close.Date())
	s.Closings = append(s.Closings, close)
}

// AddPrice adds a price directive.
func (b *Builder) AddPrice(p *model.Price) {
	s := b.getOrCreate(p.Date())
	s.Prices = append(s.Prices, p)
}

// AddAssertion adds an assertion directive.
func (b *Builder) AddAssertion(a *model.Assertion) {
	if !b.accountFilter.MatchString(a.Account.String()) {
		return
	}
	if !b.commodityFilter.MatchString(a.Commodity.String()) {
		return
	}
	s := b.getOrCreate(a.Date())
	s.Assertions = append(s.Assertions, a)
}

// AddValue adds an value directive.
func (b *Builder) AddValue(a *model.Value) {
	if !b.accountFilter.MatchString(a.Account.String()) {
		return
	}
	if !b.commodityFilter.MatchString(a.Commodity.String()) {
		return
	}
	s := b.getOrCreate(a.Date())
	s.Values = append(s.Values, a)
}
