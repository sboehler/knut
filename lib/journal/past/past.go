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
	"time"

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

// Day represents a day of activity in the processed AST.
type Day struct {
	AST          *ast.Day
	Date         time.Time
	Transactions []*ast.Transaction
	Amounts      Amounts
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

// Minus mutably subtracts.
func (am Amounts) Minus(a Amounts) {
	for ca, v := range a {
		am[ca] = am[ca].Sub(v)
	}
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
