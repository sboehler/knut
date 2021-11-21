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
	"bytes"
	"fmt"
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

// Update updates the balance with the given Day
func (b *Balance) Update(day *ledger.Day, np prices.NormalizedPrices, close bool) error {

	// update date
	b.Date = day.Date
	b.NormalizedPrices = np

	var ao AccountOpener
	if err := ao.Process(b, day); err != nil {
		return err
	}

	var tb TransactionBooker
	if err := tb.Process(b, day); err != nil {
		return err
	}

	var vb ValueBooker
	if err := vb.Process(b, day); err != nil {
		return err
	}

	var ba Asserter
	if err := ba.Process(b, day); err != nil {
		return err
	}

	var tv TransactionValuator
	if err := tv.Process(b, day); err != nil {
		return err
	}

	var vtc ValuationTransactionComputer
	if err := vtc.Process(b, day); err != nil {
		return err
	}

	if close {
		var pc PeriodCloser
		if err := pc.Process(b, day); err != nil {
			return err
		}
	}

	var ac AccountCloser
	if err := ac.Process(b, day); err != nil {
		return err
	}
	return nil
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

// Options has options for processing a ledger

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
		b bytes.Buffer
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
	Period      *date.Period
	Last        int
	Valuation   *ledger.Commodity
	Close, Diff bool
}

// Build builds a sequence of balances.
func (b Builder) Build(l ledger.Ledger) ([]*Balance, error) {
	var (
		bal    = New(l.Context, b.Valuation)
		dates  = b.createDateSeries(l)
		ps     = make(prices.Prices)
		result []*Balance
		index  int
		close  bool
		np     prices.NormalizedPrices
	)
	for _, date := range dates {
		for ; index < len(l.Days); index++ {
			var step = l.Days[index]
			if step.Date.After(date) {
				break
			}
			if b.Valuation != nil {
				for _, p := range step.Prices {
					ps.Insert(p)
				}
				np = ps.Normalize(b.Valuation)
			}
			if err := bal.Update(step, np, close); err != nil {
				return nil, err
			}
			close = false
		}
		var balCopy = bal.Copy()
		balCopy.Date = date
		result = append(result, balCopy)
		close = b.Close
	}
	if b.Diff {
		result = Diffs(result)
	}
	if b.Last > 0 && b.Last < len(result) {
		result = result[len(result)-b.Last:]
	}
	return result, nil
}

func (b Builder) createDateSeries(l ledger.Ledger) []time.Time {
	var from, to time.Time
	if b.From != nil {
		from = *b.From
	} else if d, ok := l.MinDate(); ok {
		from = d
	} else {
		return nil
	}
	if b.To != nil {
		to = *b.To
	} else if d, ok := l.MaxDate(); ok {
		to = d
	} else {
		return nil
	}
	if b.Period != nil {
		return date.Series(from, to, *b.Period)
	}
	return []time.Time{from, to}
}

// Accounts keeps track of open accounts.
type Accounts map[*ledger.Account]bool

// Open opens an account.
func (oa Accounts) Open(a *ledger.Account) error {
	if oa[a] {
		return fmt.Errorf("account %v is already open", a)
	}
	oa[a] = true
	return nil
}

// Close closes an account.
func (oa Accounts) Close(a *ledger.Account) error {
	if !oa[a] {
		return fmt.Errorf("account %v is already closed", a)
	}
	delete(oa, a)
	return nil
}

// IsOpen returns whether an account is open.
func (oa Accounts) IsOpen(a *ledger.Account) bool {
	if oa[a] {
		return true
	}
	return a.Type() == ledger.EQUITY
}

// Copy copies accounts.
func (oa Accounts) Copy() Accounts {
	var res = make(map[*ledger.Account]bool, len(oa))
	for a := range oa {
		res[a] = true
	}
	return res
}
