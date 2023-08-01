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

package model

import (
	"strings"
)

// Registry has context for the model, namely a collection of
// referenced accounts and commodities.
type Registry struct {
	accounts    *Accounts
	commodities *Commodities
}

// NewContext creates a new, empty context.
func NewContext() Registry {
	return Registry{
		accounts:    NewAccounts(),
		commodities: NewCommodities(),
	}
}

// GetAccount returns an account.
func (ctx Registry) GetAccount(name string) (*Account, error) {
	return ctx.accounts.Get(name)
}

// Account returns a commodity or panics.
func (ctx Registry) Account(name string) *Account {
	c, err := ctx.GetAccount(name)
	if err != nil {
		panic(err)
	}
	return c
}

// GetCommodity returns a commodity.
func (ctx Registry) GetCommodity(name string) (*Commodity, error) {
	return ctx.commodities.Get(name)
}

// Commodity returns a commodity or panics.
func (ctx Registry) Commodity(name string) *Commodity {
	c, err := ctx.GetCommodity(name)
	if err != nil {
		panic(err)
	}
	return c
}

// TBDAccount returns the TBD account.
func (ctx Registry) TBDAccount() *Account {
	return ctx.Account("Expenses:TBD")
}

// ValuationAccountFor returns the valuation account which corresponds to
// the given Asset or Liability account.
func (ctx Registry) ValuationAccountFor(a *Account) *Account {
	suffix := a.Split()[1:]
	segments := append(ctx.Account("Income").Split(), suffix...)
	return ctx.Account(strings.Join(segments, ":"))
}

// Accounts returns the accounts.
func (ctx Registry) Accounts() *Accounts {
	return ctx.accounts
}

// Commodities returns the commodities.
func (ctx Registry) Commodities() *Commodities {
	return ctx.commodities
}
