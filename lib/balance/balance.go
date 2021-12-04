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

package balance

import (
	"fmt"
	"strings"
	"time"

	"github.com/sboehler/knut/lib/ledger"
	"github.com/sboehler/knut/lib/prices"
	"github.com/sboehler/knut/lib/printer"

	"github.com/shopspring/decimal"
)

// Balance represents a balance for accounts at the given date.
type Balance struct {
	Date             time.Time
	Amounts, Values  map[*ledger.Account]map[*ledger.Commodity]decimal.Decimal
	Accounts         Accounts
	Context          ledger.Context
	Valuation        *ledger.Commodity
	NormalizedPrices prices.NormalizedPrices
}

// New creates a new balance.
func New(ctx ledger.Context, valuation *ledger.Commodity) *Balance {
	return &Balance{
		Context:   ctx,
		Amounts:   make(map[*ledger.Account]map[*ledger.Commodity]decimal.Decimal),
		Values:    make(map[*ledger.Account]map[*ledger.Commodity]decimal.Decimal),
		Accounts:  make(Accounts),
		Valuation: valuation,
	}
}

// Copy deeply copies the balance
func (b *Balance) Copy() *Balance {
	var nb = New(b.Context, b.Valuation)
	nb.Date = b.Date
	nb.NormalizedPrices = b.NormalizedPrices
	for pos, amt := range b.Amounts {
		nb.Amounts[pos] = amt
	}
	for pos, val := range b.Values {
		nb.Values[pos] = val
	}
	nb.Accounts = b.Accounts.Copy()
	return nb
}

// Minus mutably subtracts the given balance from the receiver.
func (b *Balance) Minus(bo *Balance) {
	for acc, cm := range bo.Amounts {
		for com, va := range cm {
			vs, ok := b.Amounts[acc]
			if !ok {
				vs = make(map[*ledger.Commodity]decimal.Decimal)
				b.Amounts[acc] = vs
			}
			vs[com] = vs[com].Sub(va)
		}
	}
	for acc, cm := range bo.Values {
		for com, va := range cm {
			vs, ok := b.Values[acc]
			if !ok {
				vs = make(map[*ledger.Commodity]decimal.Decimal)
				b.Values[acc] = vs
			}
			vs[com] = vs[com].Sub(va)
		}
	}
}

func (b *Balance) bookAmount(t ledger.Transaction) error {
	for _, posting := range t.Postings {
		if !b.Accounts.IsOpen(posting.Credit) {
			return Error{t, fmt.Sprintf("credit account %s is not open", posting.Credit)}
		}
		if !b.Accounts.IsOpen(posting.Debit) {
			return Error{t, fmt.Sprintf("debit account %s is not open", posting.Debit)}
		}
		if _, ok := b.Amounts[posting.Credit]; !ok {
			b.Amounts[posting.Credit] = make(map[*ledger.Commodity]decimal.Decimal)
		}
		if _, ok := b.Amounts[posting.Debit]; !ok {
			b.Amounts[posting.Debit] = make(map[*ledger.Commodity]decimal.Decimal)
		}
		b.Amounts[posting.Credit][posting.Commodity] = b.Amounts[posting.Credit][posting.Commodity].Sub(posting.Amount)
		b.Amounts[posting.Debit][posting.Commodity] = b.Amounts[posting.Debit][posting.Commodity].Add(posting.Amount)
	}
	return nil
}

func (b *Balance) bookValue(t ledger.Transaction) error {
	for _, posting := range t.Postings {
		if _, ok := b.Values[posting.Credit]; !ok {
			b.Values[posting.Credit] = make(map[*ledger.Commodity]decimal.Decimal)
		}
		if _, ok := b.Values[posting.Debit]; !ok {
			b.Values[posting.Debit] = make(map[*ledger.Commodity]decimal.Decimal)
		}
		b.Values[posting.Credit][posting.Commodity] = b.Values[posting.Credit][posting.Commodity].Sub(posting.Value)
		b.Values[posting.Debit][posting.Commodity] = b.Values[posting.Debit][posting.Commodity].Add(posting.Value)
	}
	return nil
}

// Diffs creates the difference balances for the given
// slice of balances. The returned slice is one element smaller
// than the input slice. The balances are mutated.
func Diffs(bals []*Balance) []*Balance {
	for i := len(bals) - 1; i > 0; i-- {
		bals[i].Minus(bals[i-1])
	}
	return bals[1:]
}

// Error is an error.
type Error struct {
	directive ledger.Directive
	msg       string
}

func (be Error) Error() string {
	var (
		p printer.Printer
		b strings.Builder
	)
	fmt.Fprintf(&b, "%s:\n", be.directive.Position().Start)
	p.PrintDirective(&b, be.directive)
	fmt.Fprintf(&b, "\n%s\n", be.msg)
	return b.String()
}
