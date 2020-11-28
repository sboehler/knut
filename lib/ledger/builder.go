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
	options Options
	steps   map[time.Time]*Step
}

// Options represents configuration options for creating a ledger.
type Options struct {
	AccountsFilter    string
	CommoditiesFilter string
}

// NewBuilder creates a new builder.
func NewBuilder(options Options) *Builder {
	return &Builder{
		options: options,
		steps:   make(map[time.Time]*Step),
	}
}

// Process creates a new ledger from the results channel.
func (b *Builder) Process(results <-chan interface{}) error {
	var err error
	for res := range results {
		switch t := res.(type) {
		case error:
			return t
		case *model.Open:
			b.AddOpening(t)
		case *model.Price:
			b.AddPrice(t)
		case *model.Transaction:
			err = b.AddTransaction(t)
		case *model.Assertion:
			err = b.AddAssertion(t)
		case *model.Value:
			err = b.AddValue(t)
		case *model.Close:
			b.AddClosing(t)
		default:
			err = fmt.Errorf("Unknown: %v", t)
		}
		if err != nil {
			return err
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
func (b *Builder) AddTransaction(t *model.Transaction) error {
	var (
		matchedCredit, matchedDebit, matchedCommodity bool
		err                                           error
	)
	for _, p := range t.Postings {
		if matchedCredit, err = regexp.MatchString(b.options.AccountsFilter, p.Credit.String()); err != nil {
			return err
		}
		if matchedDebit, err = regexp.MatchString(b.options.AccountsFilter, p.Debit.String()); err != nil {
			return err
		}
		if matchedCommodity, err = regexp.MatchString(b.options.CommoditiesFilter, p.Commodity.String()); err != nil {
			return err
		}
		if (matchedCredit || matchedDebit) && matchedCommodity {
			s := b.getOrCreate(t.Date())
			s.Transactions = append(s.Transactions, t)
			break
		}
	}
	return nil
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
func (b *Builder) AddAssertion(a *model.Assertion) error {
	matched, err := regexp.MatchString(b.options.AccountsFilter, a.Account.String())
	if !matched || err != nil {
		return err
	}
	matched, err = regexp.MatchString(b.options.CommoditiesFilter, a.Commodity.String())
	if !matched || err != nil {
		return err
	}
	s := b.getOrCreate(a.Date())
	s.Assertions = append(s.Assertions, a)
	return nil
}

// AddValue adds an value directive.
func (b *Builder) AddValue(a *model.Value) error {
	matched, err := regexp.MatchString(b.options.AccountsFilter, a.Account.String())
	if !matched || err != nil {
		return err
	}
	matched, err = regexp.MatchString(b.options.CommoditiesFilter, a.Commodity.String())
	if !matched || err != nil {
		return err
	}
	s := b.getOrCreate(a.Date())
	s.Values = append(s.Values, a)
	return nil
}
