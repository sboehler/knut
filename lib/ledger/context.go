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

package ledger

import (
	"fmt"
	"strings"
)

// Context has context for this ledger, namely a collection of
// referenced accounts and
type Context struct {
	accounts    *Accounts
	commodities *Commodities
}

// NewContext creates a new, empty context.
func NewContext() Context {
	return Context{
		accounts:    NewAccounts(),
		commodities: NewCommodities(),
	}
}

// GetAccount returns an account.
func (c Context) GetAccount(name string) (*Account, error) {
	return c.accounts.Get(name)
}

// GetCommodity returns a commodity.
func (c Context) GetCommodity(name string) (*Commodity, error) {
	return c.commodities.Get(name)
}

// ValuationAccount returns the account for automatic valuation bookings.
func (c Context) ValuationAccount() *Account {
	res, err := c.accounts.Get("Equity:Valuation")
	if err != nil {
		panic(fmt.Sprintf("could not create valuation account: %v", err))
	}
	return res
}

// EquityAccount is the equity account used for trades
func (c Context) EquityAccount() *Account {
	res, err := c.accounts.Get("Equity:Equity")
	if err != nil {
		panic(fmt.Sprintf("could not create equityAccount: %v", err))
	}
	return res
}

// RetainedEarningsAccount returns the account for automatic valuation bookings.
func (c Context) RetainedEarningsAccount() *Account {
	res, err := c.accounts.Get("Equity:RetainedEarnings")
	if err != nil {
		panic(fmt.Sprintf("could not create valuationAccount: %v", err))
	}
	return res
}

// TBDAccount returns the TBD account.
func (c Context) TBDAccount() *Account {
	tbdAccount, err := c.accounts.Get("Expenses:TBD")
	if err != nil {
		panic(fmt.Sprintf("could not create Expenses:TBD account: %v", err))
	}
	return tbdAccount
}

// ValuationAccountFor returns the valuation account which corresponds to
// the given Asset or Liability account.
func (c Context) ValuationAccountFor(a *Account) (*Account, error) {
	suffix := a.Split()[1:]
	segments := append(c.ValuationAccount().Split(), suffix...)
	return c.GetAccount(strings.Join(segments, ":"))
}
