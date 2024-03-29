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

// New creates a new, empty context.
func New() *Registry {
	return &Registry{
		accounts:    account.NewRegistry(),
		commodities: commodity.NewCommodities(),
	}
}

// Accounts returns the accounts.
func (reg Registry) Accounts() *account.Registry {
	return reg.accounts
}

// Commodities returns the commodities.
func (reg Registry) Commodities() *commodity.Registry {
	return reg.commodities
}
