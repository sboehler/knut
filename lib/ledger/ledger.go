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
	"time"

	"github.com/shopspring/decimal"

	"github.com/sboehler/knut/lib/amount"
	"github.com/sboehler/knut/lib/date"
	"github.com/sboehler/knut/lib/model"
	"github.com/sboehler/knut/lib/model/accounts"
	"github.com/sboehler/knut/lib/model/commodities"
)

// Day groups all commands for a given date.
type Day struct {
	Date         time.Time
	Prices       []*Price
	Assertions   []*Assertion
	Values       []*Value
	Openings     []*Open
	Transactions []*Transaction
	Closings     []*Close
}

// Ledger is a ledger.
type Ledger []*Day

// MinDate returns the minimum date for this ledger, as the first
// date on which an account is opened (ignoring prices, for example).
func (l Ledger) MinDate() (time.Time, bool) {
	for _, s := range l {
		if len(s.Openings) > 0 {
			return s.Date, true
		}
	}
	return time.Time{}, false
}

// MaxDate returns the maximum date for the given ledger.
func (l Ledger) MaxDate() (time.Time, bool) {
	if len(l) == 0 {
		return time.Time{}, false
	}
	return l[len(l)-1].Date, true
}

// Open represents an open command.
type Open struct {
	Pos     model.Range
	Date    time.Time
	Account *accounts.Account
}

// Position returns the position.
func (o Open) Position() model.Range {
	return o.Pos
}

// Close represents a close command.
type Close struct {
	Pos     model.Range
	Date    time.Time
	Account *accounts.Account
}

// Position returns the position.
func (c Close) Position() model.Range {
	return c.Pos
}

// Posting represents a posting.
type Posting struct {
	Amount        amount.Amount
	Credit, Debit *accounts.Account
	Commodity     *commodities.Commodity
	Lot           *Lot
}

// NewPosting creates a new posting from the given parameters. If amount is negative, it
// will be inverted and the accounts reversed.
func NewPosting(crAccount, drAccount *accounts.Account, commodity *commodities.Commodity, amt decimal.Decimal) *Posting {
	if amt.IsNegative() {
		crAccount, drAccount = drAccount, crAccount
		amt = amt.Neg()
	}
	return &Posting{
		Credit:    crAccount,
		Debit:     drAccount,
		Amount:    amount.New(amt, decimal.Zero),
		Commodity: commodity,
	}
}

// Lot represents a lot.
type Lot struct {
	Date      time.Time
	Label     string
	Price     float64
	Commodity *commodities.Commodity
}

// Tag represents a tag for a transaction or booking.
type Tag string

// Transaction represents a transaction.
type Transaction struct {
	Pos         model.Range
	Date        time.Time
	Description string
	Tags        []Tag
	Postings    []*Posting
}

// Position returns the Position.
func (t Transaction) Position() model.Range {
	return t.Pos
}

// Price represents a price command.
type Price struct {
	Pos       model.Range
	Date      time.Time
	Commodity *commodities.Commodity
	Target    *commodities.Commodity
	Price     decimal.Decimal
}

// Position returns the model.Range.
func (p Price) Position() model.Range {
	return p.Pos
}

// Include represents an include directive.
type Include struct {
	Pos  model.Range
	Date time.Time
	Path string
}

// Position returns the model.Range.
func (i Include) Position() model.Range {
	return i.Pos
}

// Assertion represents a balance assertion.
type Assertion struct {
	Pos       model.Range
	Date      time.Time
	Account   *accounts.Account
	Amount    decimal.Decimal
	Commodity *commodities.Commodity
}

// Position returns the model.Range.
func (a Assertion) Position() model.Range {
	return a.Pos
}

// Value represents a value directive.
type Value struct {
	Pos       model.Range
	Date      time.Time
	Account   *accounts.Account
	Amount    decimal.Decimal
	Commodity *commodities.Commodity
}

// Position returns the model.Range.
func (v Value) Position() model.Range {
	return v.Pos
}

// Accrual represents an accrual.
type Accrual struct {
	Pos         model.Range
	Period      date.Period
	T0, T1      time.Time
	Account     *accounts.Account
	Transaction *Transaction
}

// Position returns the position.
func (a *Accrual) Position() model.Range {
	return a.Pos
}

// Expand expands an accrual transaction.
func (a *Accrual) Expand() ([]*Transaction, error) {
	t := a.Transaction
	if l := len(t.Postings); l != 1 {
		return nil, fmt.Errorf("%s: accrual expansion: number of postings is %d, must be 1", a.Transaction.Position().Start, l)
	}
	posting := t.Postings[0]
	var (
		crAccountSingle, drAccountSingle, crAccountMulti, drAccountMulti = a.Account, a.Account, a.Account, a.Account
	)
	switch {
	case isAL(posting.Credit) && isIE(posting.Debit):
		crAccountSingle = posting.Credit
		drAccountMulti = posting.Debit
	case isIE(posting.Credit) && isAL(posting.Debit):
		crAccountMulti = posting.Credit
		drAccountSingle = posting.Debit
	case isIE(posting.Credit) && isIE(posting.Debit):
		crAccountMulti = posting.Credit
		drAccountMulti = posting.Debit
	default:
		crAccountSingle = posting.Credit
		drAccountSingle = posting.Debit
	}
	amount := posting.Amount.Amount()
	dates := date.Series(a.T0, a.T1, a.Period)[1:]
	rated, rem := amount.QuoRem(decimal.NewFromInt(int64(len(dates))), 1)

	var result []*Transaction
	if crAccountMulti != drAccountMulti {
		for i, date := range dates {
			a := rated
			if i == 0 {
				a = a.Add(rem)
			}
			result = append(result, &Transaction{
				Pos:         t.Pos,
				Date:        date,
				Tags:        t.Tags,
				Description: fmt.Sprintf("%s (accrual %d/%d)", t.Description, i+1, len(dates)),
				Postings: []*Posting{
					NewPosting(crAccountMulti, drAccountMulti, posting.Commodity, a),
				},
			})
		}
	}
	if crAccountSingle != drAccountSingle {
		result = append(result, &Transaction{
			Pos:         t.Pos,
			Date:        t.Date,
			Tags:        t.Tags,
			Description: t.Description,
			Postings: []*Posting{
				NewPosting(crAccountSingle, drAccountSingle, posting.Commodity, posting.Amount.Amount()),
			},
		})

	}
	return result, nil
}

func isAL(a *accounts.Account) bool {
	return a.Type() == accounts.ASSETS || a.Type() == accounts.LIABILITIES
}

func isIE(a *accounts.Account) bool {
	return a.Type() == accounts.INCOME || a.Type() == accounts.EXPENSES
}
