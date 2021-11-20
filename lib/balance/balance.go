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
	"strings"
	"time"

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
	Amounts, Values  map[CommodityAccount]decimal.Decimal
	openAccounts     Accounts
	accounts         *accounts.Accounts
	Valuation        *commodities.Commodity
	NormalizedPrices prices.NormalizedPrices
}

// New creates a new balance.
func New(valuation *commodities.Commodity, accs *accounts.Accounts) *Balance {
	return &Balance{
		accounts:     accs,
		Amounts:      make(map[CommodityAccount]decimal.Decimal),
		Values:       make(map[CommodityAccount]decimal.Decimal),
		openAccounts: make(Accounts),
		Valuation:    valuation,
	}
}

// Copy deeply copies the balance
func (b *Balance) Copy() *Balance {
	var nb = New(b.Valuation, b.accounts)
	nb.Date = b.Date
	nb.NormalizedPrices = b.NormalizedPrices
	for pos, amt := range b.Amounts {
		nb.Amounts[pos] = amt
	}
	for pos, val := range b.Values {
		nb.Values[pos] = val
	}
	nb.openAccounts = b.openAccounts.Copy()
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
	var err error

	// update date
	b.Date = day.Date
	b.NormalizedPrices = np

	// open accounts
	for _, o := range day.Openings {
		if err := b.openAccounts.Open(o.Account); err != nil {
			return err
		}
	}

	// book journal transaction amounts
	for _, t := range day.Transactions {
		if err = b.bookTransactionAmounts(t); err != nil {
			return err
		}
	}

	// create and book value transactions
	for _, v := range day.Values {
		var t ledger.Transaction
		if t, err = b.processValue(v); err != nil {
			return err
		}
		if err = b.bookTransactionAmounts(t); err != nil {
			return err
		}
		day.Transactions = append(day.Transactions, t)
	}

	// process balance assertions
	for _, a := range day.Assertions {
		if err := b.processBalanceAssertion(a); err != nil {
			return err
		}
	}

	// valuate transactions and book transaction values
	for _, t := range day.Transactions {
		if err := b.valuateTransaction(t); err != nil {
			return err
		}
		if err := b.bookTransactionValues(t); err != nil {
			return err
		}
	}

	// compute and append valuation transactions
	var valTrx []ledger.Transaction
	if valTrx, err = b.computeValuationTransactions(); err != nil {
		return err
	}
	for _, t := range valTrx {
		if err := b.bookTransactionValues(t); err != nil {
			return err
		}
	}
	day.Transactions = append(day.Transactions, valTrx...)

	// close income and expense accounts if necessary
	if close {
		var closingTransactions = b.computeClosingTransactions()
		day.Transactions = append(day.Transactions, closingTransactions...)
		for _, t := range closingTransactions {
			if err := b.bookTransactionAmounts(t); err != nil {
				return err
			}
			if err := b.bookTransactionValues(t); err != nil {
				return err
			}
		}
	}

	// close accounts
	for _, c := range day.Closings {
		for pos, amount := range b.Amounts {
			if pos.Account != c.Account {
				continue
			}
			if !amount.IsZero() || !b.Values[pos].IsZero() {
				return Error{c, "account has nonzero position"}
			}
			delete(b.Amounts, pos)
			delete(b.Values, pos)
		}
		if err := b.openAccounts.Close(c.Account); err != nil {
			return err
		}
	}
	return nil
}

func (b *Balance) bookTransactionAmounts(t ledger.Transaction) error {
	for _, posting := range t.Postings {
		if !b.openAccounts.IsOpen(posting.Credit) {
			return Error{t, fmt.Sprintf("credit account %s is not open", posting.Credit)}
		}
		if !b.openAccounts.IsOpen(posting.Debit) {
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

func (b *Balance) bookTransactionValues(t ledger.Transaction) error {
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

func (b *Balance) computeClosingTransactions() []ledger.Transaction {
	var result []ledger.Transaction
	for pos, va := range b.Amounts {
		var at = pos.Account.Type()
		if at != accounts.INCOME && at != accounts.EXPENSES {
			continue
		}
		result = append(result, ledger.Transaction{
			Date:        b.Date,
			Description: fmt.Sprintf("Closing %v to retained earnings", pos),
			Tags:        nil,
			Postings: []ledger.Posting{
				{
					Amount:    va,
					Value:     b.Values[pos],
					Commodity: pos.Commodity,
					Credit:    pos.Account,
					Debit:     b.accounts.RetainedEarningsAccount(),
				},
			},
		})
	}
	return result
}

var descCache = make(map[CommodityAccount]string)

// computeValuationTransactions checks whether the valuation for the positions
// corresponds to the amounts. If not, the difference is due to a valuation
// change of the previous amount, and a transaction is created to adjust the
// valuation.
func (b *Balance) computeValuationTransactions() ([]ledger.Transaction, error) {
	if b.Valuation == nil {
		return nil, nil
	}
	var result []ledger.Transaction
	for pos, va := range b.Amounts {
		if pos.Commodity == b.Valuation {
			continue
		}
		var at = pos.Account.Type()
		if at != accounts.ASSETS && at != accounts.LIABILITIES {
			continue
		}
		value, err := b.NormalizedPrices.Valuate(pos.Commodity, va)
		if err != nil {
			panic(fmt.Sprintf("no valuation found for commodity %s", pos.Commodity))
		}
		var diff = value.Sub(b.Values[pos])
		if diff.IsZero() {
			continue
		}
		var desc string
		if s, ok := descCache[pos]; ok {
			desc = s
		} else {
			desc = fmt.Sprintf("Adjust value of %s in account %s", pos.Commodity, pos.Account)
			descCache[pos] = desc
		}
		valAcc, err := b.valuationAccountFor(pos.Account)
		if err != nil {
			panic(fmt.Sprintf("could not obtain valuation account for account %s", pos.Account))
		}
		// create a transaction to adjust the valuation
		result = append(result, ledger.Transaction{
			Date:        b.Date,
			Description: desc,
			Postings: []ledger.Posting{
				{
					Value:     diff,
					Credit:    valAcc,
					Debit:     pos.Account,
					Commodity: pos.Commodity,
				},
			},
		})
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

func (b Balance) valuationAccountFor(a *accounts.Account) (*accounts.Account, error) {
	suffix := a.Split()[1:]
	segments := append(b.accounts.ValuationAccount().Split(), suffix...)
	return b.accounts.Get(strings.Join(segments, ":"))
}

func (b *Balance) valuateTransaction(t ledger.Transaction) error {
	if b.Valuation == nil {
		return nil
	}
	for i := range t.Postings {
		var posting = &t.Postings[i]
		if b.Valuation == posting.Commodity {
			posting.Value = posting.Amount
			continue
		}
		var err error
		if posting.Value, err = b.NormalizedPrices.Valuate(posting.Commodity, posting.Amount); err != nil {
			return Error{t, fmt.Sprintf("no price found for commodity %s", posting.Commodity)}
		}
	}
	return nil
}

func (b *Balance) processValue(v ledger.Value) (ledger.Transaction, error) {
	if !b.openAccounts.IsOpen(v.Account) {
		return ledger.Transaction{}, Error{v, "account is not open"}
	}
	valAcc, err := b.valuationAccountFor(v.Account)
	if err != nil {
		return ledger.Transaction{}, err
	}
	var pos = CommodityAccount{v.Account, v.Commodity}
	return ledger.Transaction{
		Date:        v.Date,
		Description: fmt.Sprintf("Valuation adjustment for %v", pos),
		Tags:        nil,
		Postings: []ledger.Posting{
			ledger.NewPosting(valAcc, v.Account, pos.Commodity, v.Amount.Sub(b.Amounts[pos])),
		},
	}, nil
}

func (b *Balance) processBalanceAssertion(a ledger.Assertion) error {
	if !b.openAccounts.IsOpen(a.Account) {
		return Error{a, "account is not open"}
	}
	var pos = CommodityAccount{a.Account, a.Commodity}
	va, ok := b.Amounts[pos]
	if !ok || !va.Equal(a.Amount) {
		return Error{a, fmt.Sprintf("assertion failed: account %s has %s %s", a.Account, va, pos.Commodity)}
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
		bal    = New(b.Valuation, l.Accounts)
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

// Accounts keeps track of accounts.
type Accounts map[*accounts.Account]bool

// Open opens an account.
func (oa Accounts) Open(a *accounts.Account) error {
	if oa[a] {
		return fmt.Errorf("account %v is already open", a)
	}
	oa[a] = true
	return nil
}

// Close closes an account.
func (oa Accounts) Close(a *accounts.Account) error {
	if !oa[a] {
		return fmt.Errorf("account %v is already closed", a)
	}
	delete(oa, a)
	return nil
}

// IsOpen returns whether an account is open.
func (oa Accounts) IsOpen(a *accounts.Account) bool {
	if oa[a] {
		return true
	}
	return a.Type() == accounts.EQUITY
}

// Copy copies accounts.
func (oa Accounts) Copy() Accounts {
	var res = make(map[*accounts.Account]bool, len(oa))
	for a := range oa {
		res[a] = true
	}
	return res
}
