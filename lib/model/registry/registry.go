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

package registry

import (
	"strings"

	"github.com/sboehler/knut/lib/model/account"
	"github.com/sboehler/knut/lib/model/commodity"
)

type Account = account.Account
type Commodity = commodity.Commodity

// Registry has context for the model, namely a collection of
// referenced accounts and commodities.
type Registry struct {
	accounts    *account.Registry
	commodities *commodity.Registry
}

// NewContext creates a new, empty context.
func NewContext() *Registry {
	return &Registry{
		accounts:    account.NewRegistry(),
		commodities: commodity.NewCommodities(),
	}
}

// GetAccount returns an account.
func (reg Registry) GetAccount(name string) (*Account, error) {
	return reg.accounts.Get(name)
}

// Account returns a commodity or panics.
func (reg Registry) Account(name string) *Account {
	c, err := reg.GetAccount(name)
	if err != nil {
		panic(err)
	}
	return c
}

// GetCommodity returns a commodity.
func (reg Registry) GetCommodity(name string) (*Commodity, error) {
	return reg.commodities.Get(name)
}

// Commodity returns a commodity or panics.
func (reg Registry) Commodity(name string) *Commodity {
	c, err := reg.GetCommodity(name)
	if err != nil {
		panic(err)
	}
	return c
}

// TBDAccount returns the TBD account.
func (reg Registry) TBDAccount() *Account {
	return reg.Account("Expenses:TBD")
}

// ValuationAccountFor returns the valuation account which corresponds to
// the given Asset or Liability account.
func (reg Registry) ValuationAccountFor(a *Account) *Account {
	suffix := a.Split()[1:]
	segments := append(reg.Account("Income").Split(), suffix...)
	return reg.Account(strings.Join(segments, ":"))
}

// Accounts returns the accounts.
func (reg Registry) Accounts() *account.Registry {
	return reg.accounts
}

// Commodities returns the commodities.
func (reg Registry) Commodities() *commodity.Registry {
	return reg.commodities
}
