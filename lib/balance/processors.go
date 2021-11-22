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
	"sort"
	"time"

	"github.com/sboehler/knut/lib/date"
	"github.com/sboehler/knut/lib/ledger"
	"github.com/sboehler/knut/lib/prices"
)

// DateUpdater keeps track of open accounts.
type DateUpdater struct {
	Balance *Balance
}

var _ ledger.Processor = (*DateUpdater)(nil)

// Process implements Processor.
func (a DateUpdater) Process(d *ledger.Day) error {
	a.Balance.Date = d.Date
	return nil
}

// Snapshotter keeps track of open accounts.
type Snapshotter struct {
	Balance  *Balance
	From, To *time.Time
	Last     int
	Diff     bool
	Period   date.Period
	Result   *[]*Balance
	dates    []time.Time
	index    int
}

var (
	_ ledger.Initializer = (*Snapshotter)(nil)
	_ ledger.Processor     = (*Snapshotter)(nil)
	_ ledger.Finalizer   = (*Snapshotter)(nil)
)

// Initialize implements Initializer.
func (a *Snapshotter) Initialize(l ledger.Ledger) error {
	a.dates = l.Dates(a.From, a.To, a.Period)
	var offset = 0
	if a.Diff {
		offset = 1
	}
	if a.Last > 0 && a.Last < len(a.dates)-offset {
		a.dates = a.dates[len(a.dates)-a.Last-offset:]
	}
	*a.Result = make([]*Balance, len(a.dates))
	return nil
}

// Process implements Processor.
func (a *Snapshotter) Process(d *ledger.Day) error {
	if len(a.dates) == 0 || a.index >= len(a.dates) {
		return nil
	}
	for ; a.index < len(a.dates) && d.Date.After(a.dates[a.index]); a.index++ {
		var cp = a.Balance.Copy()
		cp.Date = a.dates[a.index]
		(*a.Result)[a.index] = cp
	}
	return nil
}

// Finalize implements Finalizer.
func (a *Snapshotter) Finalize() error {
	for ; a.index < len(a.dates); a.index++ {
		var cp = a.Balance.Copy()
		cp.Date = a.dates[a.index]
		(*a.Result)[a.index] = cp
	}
	if a.Diff {
		*a.Result = Diffs(*a.Result)
	}
	return nil
}

// PriceUpdater keeps track of prices.
type PriceUpdater struct {
	Balance *Balance
	prices  prices.Prices
}

var (
	_ ledger.Initializer = (*PriceUpdater)(nil)
	_ ledger.Processor     = (*PriceUpdater)(nil)
)

// Initialize implements Initializer.
func (a *PriceUpdater) Initialize(_ ledger.Ledger) error {
	a.prices = make(prices.Prices)
	return nil
}

// Process implements Processor.
func (a *PriceUpdater) Process(d *ledger.Day) error {
	if a.Balance.Valuation == nil {
		return nil
	}
	for _, p := range d.Prices {
		a.prices.Insert(p)
	}
	a.Balance.NormalizedPrices = a.prices.Normalize(a.Balance.Valuation)
	return nil
}

// AccountOpener keeps track of open accounts.
type AccountOpener struct {
	Balance *Balance
}

var _ ledger.Processor = (*AccountOpener)(nil)

// Process implements Processor.
func (a AccountOpener) Process(d *ledger.Day) error {
	for _, o := range d.Openings {
		if err := a.Balance.Accounts.Open(o.Account); err != nil {
			return err
		}
	}
	return nil
}

// TransactionBooker books transaction amounts.
type TransactionBooker struct {
	Balance *Balance
}

var _ ledger.Processor = (*TransactionBooker)(nil)

// Process implements Processor.
func (tb TransactionBooker) Process(d *ledger.Day) error {
	// book journal transaction amounts
	for _, t := range d.Transactions {
		if err := tb.Balance.bookAmount(t); err != nil {
			return err
		}
	}
	return nil
}

// ValueBooker books amounts for value directives.
type ValueBooker struct {
	Balance *Balance
}

var _ ledger.Processor = (*ValueBooker)(nil)

// Process implements Processor.
func (tb ValueBooker) Process(d *ledger.Day) error {
	for _, v := range d.Values {
		var (
			t   ledger.Transaction
			err error
		)
		if t, err = tb.processValue(v); err != nil {
			return err
		}
		if err = tb.Balance.bookAmount(t); err != nil {
			return err
		}
		d.Transactions = append(d.Transactions, t)
	}
	d.Values = nil
	return nil
}

func (tb ValueBooker) processValue(v ledger.Value) (ledger.Transaction, error) {
	if !tb.Balance.Accounts.IsOpen(v.Account) {
		return ledger.Transaction{}, Error{v, "account is not open"}
	}
	valAcc, err := tb.Balance.Context.ValuationAccountFor(v.Account)
	if err != nil {
		return ledger.Transaction{}, err
	}
	var pos = CommodityAccount{v.Account, v.Commodity}
	return ledger.Transaction{
		Date:        v.Date,
		Description: fmt.Sprintf("Valuation adjustment for %v", pos),
		Tags:        nil,
		Postings: []ledger.Posting{
			ledger.NewPosting(valAcc, v.Account, pos.Commodity, v.Amount.Sub(tb.Balance.Amounts[pos])),
		},
	}, nil
}

// Asserter keeps track of open accounts.
type Asserter struct {
	Balance *Balance
}

var _ ledger.Processor = (*Asserter)(nil)

// Process implements Processor.
func (as Asserter) Process(d *ledger.Day) error {
	for _, a := range d.Assertions {
		if err := as.processBalanceAssertion(as.Balance, a); err != nil {
			return err
		}
	}
	return nil
}

func (as Asserter) processBalanceAssertion(b *Balance, a ledger.Assertion) error {
	if !b.Accounts.IsOpen(a.Account) {
		return Error{a, "account is not open"}
	}
	var pos = CommodityAccount{a.Account, a.Commodity}
	va, ok := b.Amounts[pos]
	if !ok || !va.Equal(a.Amount) {
		return Error{a, fmt.Sprintf("assertion failed: account %s has %s %s", a.Account, va, pos.Commodity)}
	}
	return nil
}

// TransactionValuator valuates transactions.
type TransactionValuator struct {
	Balance *Balance
}

var _ ledger.Processor = (*TransactionValuator)(nil)

// Process implements Processor.
func (as TransactionValuator) Process(d *ledger.Day) error {
	for _, t := range d.Transactions {
		if err := as.valuateTransaction(as.Balance, t); err != nil {
			return err
		}
		if err := as.Balance.bookValue(t); err != nil {
			return err
		}
	}
	return nil
}

func (as TransactionValuator) valuateTransaction(b *Balance, t ledger.Transaction) error {
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

// ValuationTransactionComputer valuates transactions.
type ValuationTransactionComputer struct {
	Balance *Balance
}

var _ ledger.Processor = (*ValuationTransactionComputer)(nil)

// Process implements Processor.
func (vtc ValuationTransactionComputer) Process(d *ledger.Day) error {
	valTrx, err := vtc.computeValuationTransactions(vtc.Balance)
	if err != nil {
		return err
	}
	for _, t := range valTrx {
		if err := vtc.Balance.bookValue(t); err != nil {
			return err
		}
	}
	d.Transactions = append(d.Transactions, valTrx...)

	return nil
}

var descCache = make(map[CommodityAccount]string)

// computeValuationTransactions checks whether the valuation for the positions
// corresponds to the amounts. If not, the difference is due to a valuation
// change of the previous amount, and a transaction is created to adjust the
// valuation.
func (vtc ValuationTransactionComputer) computeValuationTransactions(b *Balance) ([]ledger.Transaction, error) {
	if b.Valuation == nil {
		return nil, nil
	}
	var result []ledger.Transaction
	for pos, va := range b.Amounts {
		if pos.Commodity == b.Valuation {
			continue
		}
		var at = pos.Account.Type()
		if at != ledger.ASSETS && at != ledger.LIABILITIES {
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
		valAcc, err := b.Context.ValuationAccountFor(pos.Account)
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

// PeriodCloser closes the accounting period.
type PeriodCloser struct {
	Balance *Balance
}

var _ ledger.Processor = (*PeriodCloser)(nil)

// Process implements Processor.
func (as PeriodCloser) Process(d *ledger.Day) error {
	var closingTransactions = as.computeClosingTransactions()
	d.Transactions = append(d.Transactions, closingTransactions...)
	for _, t := range closingTransactions {
		if err := as.Balance.bookAmount(t); err != nil {
			return err
		}
		if err := as.Balance.bookValue(t); err != nil {
			return err
		}
	}
	return nil
}

func (as PeriodCloser) computeClosingTransactions() []ledger.Transaction {
	var result []ledger.Transaction
	for pos, va := range as.Balance.Amounts {
		var at = pos.Account.Type()
		if at != ledger.INCOME && at != ledger.EXPENSES {
			continue
		}
		result = append(result, ledger.Transaction{
			Date:        as.Balance.Date,
			Description: fmt.Sprintf("Closing %v to retained earnings", pos),
			Tags:        nil,
			Postings: []ledger.Posting{
				{
					Amount:    va,
					Value:     as.Balance.Values[pos],
					Commodity: pos.Commodity,
					Credit:    pos.Account,
					Debit:     as.Balance.Context.RetainedEarningsAccount(),
				},
			},
		})
	}
	return result
}

// AccountCloser closes accounts.
type AccountCloser struct {
	Balance *Balance
}

var _ ledger.Processor = (*AccountCloser)(nil)

// Process implements Processor.
func (vtc AccountCloser) Process(d *ledger.Day) error {
	for _, c := range d.Closings {
		for pos, amount := range vtc.Balance.Amounts {
			if pos.Account != c.Account {
				continue
			}
			if !amount.IsZero() || !vtc.Balance.Values[pos].IsZero() {
				return Error{c, "account has nonzero position"}
			}
			delete(vtc.Balance.Amounts, pos)
			delete(vtc.Balance.Values, pos)
		}
		if err := vtc.Balance.Accounts.Close(c.Account); err != nil {
			return err
		}
	}
	return nil
}
