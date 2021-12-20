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

package past

import (
	"fmt"
	"sort"
	"time"

	"github.com/sboehler/knut/lib/common/date"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
	"github.com/shopspring/decimal"
)

// PAST is a processed AST.
type PAST struct {
	Days    []*Day
	Context journal.Context
}

// MinDate returns the minimum date for this ledger, as the first
// date on which an account is opened (ignoring prices, for example).
func (l PAST) MinDate() (time.Time, bool) {
	for _, s := range l.Days {
		if len(s.AST.Openings) > 0 {
			return s.Date, true
		}
	}
	return time.Time{}, false
}

// MaxDate returns the maximum date for the given
func (l PAST) MaxDate() (time.Time, bool) {
	if len(l.Days) == 0 {
		return time.Time{}, false
	}
	return l.Days[len(l.Days)-1].Date, true
}

// Dates returns a series of dates which covers the first
// and last date in the ast.
func (l PAST) Dates(from, to time.Time, period date.Period) []time.Time {
	if len(l.Days) == 0 {
		return nil
	}
	var t0, t1 time.Time
	if !from.IsZero() {
		t0 = from
	} else {
		t0, _ = l.MinDate()
	}
	if !to.IsZero() {
		t1 = to
	} else {
		t1, _ = l.MaxDate()
	}
	return date.Series(t0, t1, period)
}

// ActualDates returns a series like Dates, but containing the latest available,
// actual dates from the days in the ast. That is, an element of the result
// array is either the zero date (if it is before the first date in the ledger),
// or the latest date in the ledger which is smaller or equal than the corresponding
// element in the input array.
func (l PAST) ActualDates(ds []time.Time) []time.Time {
	var actuals = make([]time.Time, 0, len(ds))
	for _, date := range ds {
		if len(l.Days) == 0 || date.Before(l.Days[0].Date) {
			// no days in the ledger, or date before all ledger days
			actuals = append(actuals, time.Time{})
			continue
		}
		index := sort.Search(len(l.Days), func(i int) bool { return !l.Days[i].Date.Before(date) })
		if index == len(l.Days) {
			// all days are after the date, use the latest one
			actuals = append(actuals, l.Days[len(l.Days)-1].Date)
			continue
		}
		actuals = append(actuals, l.Days[index].Date)
	}
	return actuals
}

// Day represents a day of activity in the processed AST.
type Day struct {
	AST          *ast.Day
	Date         time.Time
	Transactions []*ast.Transaction
	Amounts      Amounts

	// Legacy fields
	Prices     []*ast.Price
	Assertions []*ast.Assertion
	Values     []*ast.Value
	Openings   []*ast.Open
	Closings   []*ast.Close
}

// Less establishes an ordering on Day.
func (d *Day) Less(d2 *Day) bool {
	return d.Date.Before(d2.Date)
}

// CommodityAccount represents a position.
type CommodityAccount struct {
	Account   *journal.Account
	Commodity *journal.Commodity
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

// Amounts keeps track of amounts by account and commodity.
type Amounts map[CommodityAccount]decimal.Decimal

// Amount returns the amount for the given account and commodity.
func (am Amounts) Amount(a *journal.Account, c *journal.Commodity) decimal.Decimal {
	return am[CommodityAccount{Account: a, Commodity: c}]
}

// Book books the given amount.
func (am Amounts) Book(cr, dr *journal.Account, a decimal.Decimal, c *journal.Commodity) {
	var (
		crPos = CommodityAccount{cr, c}
		drPos = CommodityAccount{dr, c}
	)
	am[crPos] = am[crPos].Sub(a)
	am[drPos] = am[drPos].Add(a)
}

// Clone clones these amounts.
func (am Amounts) Clone() Amounts {
	clone := make(Amounts)
	for ca, v := range am {
		clone[ca] = v
	}
	return clone
}

// Accounts keeps track of open accounts.
type Accounts map[*journal.Account]bool

// Open opens an account.
func (oa Accounts) Open(a *journal.Account) error {
	if oa[a] {
		return fmt.Errorf("account %v is already open", a)
	}
	oa[a] = true
	return nil
}

// Close closes an account.
func (oa Accounts) Close(a *journal.Account) error {
	if !oa[a] {
		return fmt.Errorf("account %v is already closed", a)
	}
	delete(oa, a)
	return nil
}

// IsOpen returns whether an account is open.
func (oa Accounts) IsOpen(a *journal.Account) bool {
	if oa[a] {
		return true
	}
	return a.Type() == journal.EQUITY
}

// Copy copies accounts.
func (oa Accounts) Copy() Accounts {
	var res = make(map[*journal.Account]bool, len(oa))
	for a := range oa {
		res[a] = true
	}
	return res
}
