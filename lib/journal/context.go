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

package journal

import (
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
func (ctx Context) GetAccount(name string) (*Account, error) {
	return ctx.accounts.Get(name)
}

// Account returns a commodity or panics.
func (ctx Context) Account(name string) *Account {
	c, err := ctx.GetAccount(name)
	if err != nil {
		panic(err)
	}
	return c
}

// GetCommodity returns a commodity.
func (ctx Context) GetCommodity(name string) (*Commodity, error) {
	return ctx.commodities.Get(name)
}

// Commodity returns a commodity or panics.
func (ctx Context) Commodity(name string) *Commodity {
	c, err := ctx.GetCommodity(name)
	if err != nil {
		panic(err)
	}
	return c
}

// ValuationAccount returns the account for automatic valuation bookings.
func (ctx Context) ValuationAccount() *Account {
	return ctx.Account("Income:CapitalGain")
}

// TBDAccount returns the TBD account.
func (ctx Context) TBDAccount() *Account {
	return ctx.Account("Expenses:TBD")
}

// ValuationAccountFor returns the valuation account which corresponds to
// the given Asset or Liability account.
func (ctx Context) ValuationAccountFor(a *Account) *Account {
	suffix := a.Split()[1:]
	segments := append(ctx.ValuationAccount().Split(), suffix...)
	return ctx.Account(strings.Join(segments, ":"))
}

// Accounts returns the accounts.
func (ctx Context) Accounts() *Accounts {
	return ctx.accounts
}

// Commodities returns the commodities.
func (ctx Context) Commodities() *Commodities {
	return ctx.commodities
}
