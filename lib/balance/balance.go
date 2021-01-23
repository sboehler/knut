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
	"sort"
	"time"

	"github.com/sboehler/knut/lib/amount"
	"github.com/sboehler/knut/lib/date"
	"github.com/sboehler/knut/lib/ledger"
	"github.com/sboehler/knut/lib/model/accounts"
	"github.com/sboehler/knut/lib/model/commodities"
	"github.com/sboehler/knut/lib/prices"
	"github.com/sboehler/knut/lib/printer"

	"github.com/shopspring/decimal"
)

// Balance represents a balance for accounts at the given date.
type Balance struct {
	Date             time.Time
	Positions        map[CommodityAccount]amount.Amount
	Account          map[*accounts.Account]bool
	Prices           prices.Prices
	Valuation        *commodities.Commodity
	NormalizedPrices prices.NormalizedPrices
}

// New creates a new balance.
func New(valuation *commodities.Commodity) *Balance {
	return &Balance{
		Positions: make(map[CommodityAccount]amount.Amount),
		Account: map[*accounts.Account]bool{
			accounts.ValuationAccount():        true,
			accounts.RetainedEarningsAccount(): true,
		},
		Prices:    make(prices.Prices),
		Valuation: valuation,
	}
}

// Copy deeply copies the balance
func (b *Balance) Copy() *Balance {
	var nb = New(b.Valuation)
	nb.Prices = b.Prices.Copy()

	// immutable
	nb.NormalizedPrices = b.NormalizedPrices

	nb.Date = b.Date
	for pos, val := range b.Positions {
		nb.Positions[pos] = val
	}
	for acc := range b.Account {
		nb.Account[acc] = true
	}
	return nb
}

// Minus mutably subtracts the given balance from the receiver.
func (b *Balance) Minus(bo *Balance) {
	for pos, va := range bo.Positions {
		b.Positions[pos] = b.Positions[pos].Minus(va)
	}
}

// Update updates the balance with the given Day
func (b *Balance) Update(day *ledger.Day, close bool) error {

	// update date
	b.Date = day.Date

	// update prices
	for _, p := range day.Prices {
		b.Prices.Insert(p)
	}

	// update normalized prices
	if b.Valuation != nil {
		b.NormalizedPrices = b.Prices.Normalize(b.Valuation)
	}

	// open accounts
	for _, o := range day.Openings {
		if _, isOpen := b.Account[o.Account]; isOpen {
			return fmt.Errorf("Account %v is already open", o)
		}
		b.Account[o.Account] = true
	}

	// valuate and book journal transactions
	for _, t := range day.Transactions {
		if err := b.valuateTransaction(t); err != nil {
			return err
		}
		if err := b.bookTransaction(t); err != nil {
			return err
		}
	}

	// create and book value transactions
	for _, v := range day.Values {
		t, err := b.processValue(v)
		if err != nil {
			return err
		}
		day.Transactions = append(day.Transactions, t)
		if err := b.valuateTransaction(t); err != nil {
			return err
		}
		if err := b.bookTransaction(t); err != nil {
			return err
		}
	}

	// compute and append valuation transactions
	valTrx, err := b.computeValuationTransactions()
	if err != nil {
		return err
	}
	day.Transactions = append(day.Transactions, valTrx...)

	// book transactions
	for _, t := range valTrx {
		if err := b.bookTransaction(t); err != nil {
			return err
		}
	}

	// close income and expense accounts if necessary
	if close {
		var closingTransactions = b.computeClosingTransactions()
		day.Transactions = append(day.Transactions, closingTransactions...)
		for _, t := range closingTransactions {
			if err := b.bookTransaction(t); err != nil {
				return err
			}
		}
	}

	// process balance assertions
	for _, a := range day.Assertions {
		if err := b.processBalanceAssertion(a); err != nil {
			return err
		}
	}

	// close accounts
	for _, c := range day.Closings {
		if _, isOpen := b.Account[c.Account]; !isOpen {
			return Error{c, "account is not open"}
		}
		for pos, amount := range b.Positions {
			if pos.Account == c.Account && !amount.Amount().IsZero() {
				return Error{c, "account has nonzero position"}
			}
		}
		delete(b.Account, c.Account)
	}
	return nil
}

func (b *Balance) bookTransaction(t *ledger.Transaction) error {
	for _, posting := range t.Postings {
		if _, isOpen := b.Account[posting.Credit]; !isOpen {
			return Error{t, fmt.Sprintf("credit account %s is not open", posting.Credit)}
		}
		if _, isOpen := b.Account[posting.Debit]; !isOpen {
			return Error{t, fmt.Sprintf("debit account %s is not open", posting.Debit)}
		}
		crPos := CommodityAccount{posting.Credit, posting.Commodity}
		drPos := CommodityAccount{posting.Debit, posting.Commodity}
		b.Positions[crPos] = b.Positions[crPos].Minus(posting.Amount)
		b.Positions[drPos] = b.Positions[drPos].Plus(posting.Amount)
	}
	return nil
}

func (b *Balance) computeClosingTransactions() []*ledger.Transaction {
	var result []*ledger.Transaction
	for pos, va := range b.Positions {
		at := pos.Account.Type()
		if at != accounts.INCOME && at != accounts.EXPENSES {
			continue
		}
		result = append(result, &ledger.Transaction{
			Date:        b.Date,
			Description: fmt.Sprintf("Closing %v to retained earnings", pos),
			Tags:        nil,
			Postings: []*ledger.Posting{
				{
					Amount:    va,
					Commodity: pos.Commodity,
					Credit:    pos.Account,
					Debit:     accounts.RetainedEarningsAccount(),
				},
			},
		})
	}
	return result
}

// computeValuationTransactions checks whether the valuation for the positions
// corresponds to the amounts. If not, the difference is due to a valuation
// change of the previous amount, and a transaction is created to adjust the
// valuation.
func (b *Balance) computeValuationTransactions() ([]*ledger.Transaction, error) {
	if b.Valuation == nil {
		return nil, nil
	}
	var result []*ledger.Transaction
	for pos, va := range b.Positions {
		at := pos.Account.Type()
		if at != accounts.ASSETS && at != accounts.LIABILITIES {
			continue
		}
		v2, err := b.NormalizedPrices.Valuate(pos.Commodity, va.Amount())
		if err != nil {
			panic(fmt.Sprintf("no valuation found for commodity %s", pos.Commodity))
		}
		var diff = v2.Sub(va.Value())
		if !diff.IsZero() {
			// create a transaction to adjust the valuation
			result = append(result, &ledger.Transaction{
				Date:        b.Date,
				Description: fmt.Sprintf("Valuation adjustment for (%s, %s)", pos.Account, pos.Commodity),
				Tags:        nil,
				Postings: []*ledger.Posting{
					{
						Amount:    amount.New(decimal.Zero, diff),
						Credit:    accounts.ValuationAccount(),
						Debit:     pos.Account,
						Commodity: pos.Commodity,
					},
				},
			})
		}
	}
	sort.Slice(result, func(i, j int) bool {
		var p, q = result[i].Postings[0], result[j].Postings[0]
		if p.Credit != q.Credit {
			return p.Credit.String() < q.Credit.String()
		}
		if p.Debit != q.Debit {
			return p.Debit.String() < q.Debit.String()
		}
		return p.Commodity.String() < q.Commodity.String()
	})
	return result, nil
}

func (b *Balance) valuateTransaction(t *ledger.Transaction) error {
	if b.Valuation == nil {
		return nil
	}
	for _, posting := range t.Postings {
		value, err := b.NormalizedPrices.Valuate(posting.Commodity, posting.Amount.Amount())
		if err != nil {
			return Error{t, fmt.Sprintf("no price found for commodity %s", posting.Commodity)}
		}
		posting.Amount = amount.New(posting.Amount.Amount(), value)
	}
	return nil
}

func (b *Balance) processValue(v *ledger.Value) (*ledger.Transaction, error) {
	if _, isOpen := b.Account[v.Account]; !isOpen {
		return nil, Error{v, "account is not open"}
	}
	pos := CommodityAccount{v.Account, v.Commodity}
	va, ok := b.Positions[pos]
	if !ok {
		va = amount.New(decimal.Zero, decimal.Zero)
	}
	return &ledger.Transaction{
		Date:        v.Date,
		Description: fmt.Sprintf("Valuation adjustment for %v", pos),
		Tags:        nil,
		Postings: []*ledger.Posting{
			ledger.NewPosting(accounts.ValuationAccount(), v.Account, pos.Commodity, v.Amount.Sub(va.Amount())),
		},
	}, nil
}

func (b *Balance) processBalanceAssertion(a *ledger.Assertion) error {
	if _, isOpen := b.Account[a.Account]; !isOpen {
		return Error{a, "account is not open"}
	}
	var pos = CommodityAccount{a.Account, a.Commodity}
	va, ok := b.Positions[pos]
	if !ok || !va.Amount().Equal(a.Amount) {
		return Error{a, fmt.Sprintf("assertion failed: account %s has %s %s", a.Account, va.Amount(), pos.Commodity)}
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
		b bytes.Buffer
	)
	fmt.Fprintf(&b, "%s:\n", be.directive.Position().Start)
	printer.PrintDirective(&b, be.directive)
	fmt.Fprintf(&b, "\n%s\n", be.msg)
	return b.String()
}

// CommodityAccount represents a position.
type CommodityAccount struct {
	Account   *accounts.Account
	Commodity *commodities.Commodity
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
	Valuation   *commodities.Commodity
	Close, Diff bool
}

// Build builds a sequence of balances.
func (b Builder) Build(l ledger.Ledger) ([]*Balance, error) {
	var (
		bal    = New(b.Valuation)
		result []*Balance
		index  int
		close  bool
	)
	for _, date := range b.createDateSeries(l) {
		for ; index < len(l); index++ {
			if l[index].Date.After(date) {
				break
			}
			if err := bal.Update(l[index], close); err != nil {
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
