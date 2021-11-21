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
	"fmt"
	"strings"
	"time"

	"github.com/sboehler/knut/lib/date"
	"github.com/sboehler/knut/lib/ledger"
	"github.com/sboehler/knut/lib/prices"
	"github.com/sboehler/knut/lib/printer"

	"github.com/shopspring/decimal"
)

// Balance represents a balance for accounts at the given date.
type Balance struct {
	Date             time.Time
	Amounts, Values  map[CommodityAccount]decimal.Decimal
	Accounts         Accounts
	Context          ledger.Context
	Valuation        *ledger.Commodity
	NormalizedPrices prices.NormalizedPrices
}

// New creates a new balance.
func New(ctx ledger.Context, valuation *ledger.Commodity) *Balance {
	return &Balance{
		Context:   ctx,
		Amounts:   make(map[CommodityAccount]decimal.Decimal),
		Values:    make(map[CommodityAccount]decimal.Decimal),
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
	for pos, va := range bo.Amounts {
		b.Amounts[pos] = b.Amounts[pos].Sub(va)
	}
	for pos, va := range bo.Values {
		b.Values[pos] = b.Values[pos].Sub(va)
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
		var (
			crPos = CommodityAccount{posting.Credit, posting.Commodity}
			drPos = CommodityAccount{posting.Debit, posting.Commodity}
		)
		b.Amounts[crPos] = b.Amounts[crPos].Sub(posting.Amount)
		b.Amounts[drPos] = b.Amounts[drPos].Add(posting.Amount)
	}
	return nil
}

func (b *Balance) bookValue(t ledger.Transaction) error {
	for _, posting := range t.Postings {
		var (
			crPos = CommodityAccount{posting.Credit, posting.Commodity}
			drPos = CommodityAccount{posting.Debit, posting.Commodity}
		)
		b.Values[crPos] = b.Values[crPos].Sub(posting.Value)
		b.Values[drPos] = b.Values[drPos].Add(posting.Value)
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

// CommodityAccount represents a position.
type CommodityAccount struct {
	Account   *ledger.Account
	Commodity *ledger.Commodity
}

// Less establishes a partial ordering of commodity accounts.
func (p CommodityAccount) Less(p1 CommodityAccount) bool {
	if p.Account.Type() != p1.Account.Type() {
		return p.Account.Type() < p1.Account.Type()
	}
	if p.Account.String() != p1.Account.String() {
		return p.Account.String() < p1.Account.String()
	}
	return p.Commodity.String() < p1.Commodity.String()
}

// Builder builds a sequence of balances.
type Builder struct {
	From, To    *time.Time
	Period      date.Period
	Last        int
	Valuation   *ledger.Commodity
	Close, Diff bool
}

// Build builds a sequence of balances.
func (b Builder) Build(l ledger.Ledger) ([]*Balance, error) {
	var (
		balance = New(l.Context, b.Valuation)
		result  []*Balance
		steps   = []ledger.Process{
			DateUpdater{balance},
			&Snapshotter{
				Balance: balance,
				From:    b.From,
				To:      b.To,
				Period:  b.Period,
				Last:    b.Last,
				Diff:    b.Diff,
				Result:  &result},
			AccountOpener{balance},
			TransactionBooker{balance},
			ValueBooker{balance},
			Asserter{balance},
			&PriceUpdater{Balance: balance},
			TransactionValuator{balance},
			ValuationTransactionComputer{balance},
			AccountCloser{balance},
		}
	)
	if err := (ledger.Processor{Steps: steps}).Process(l); err != nil {
		return nil, err
	}
	return result, nil
}
