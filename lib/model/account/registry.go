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

package account

import (
	"fmt"
	"strings"
	"sync"
	"unicode"

	"github.com/sboehler/knut/lib/common/dict"
	"github.com/sboehler/knut/lib/common/set"
	"github.com/sboehler/knut/lib/syntax"
)

// Registry is a thread-safe collection of accounts.
type Registry struct {
	mutex    sync.RWMutex
	index    map[string]*Account
	accounts map[Type]*Account
	children map[*Account]set.Set[*Account]
	parents  map[*Account]*Account
	swaps    map[*Account]*Account
}

// NewRegistry creates a new thread-safe collection of accounts.
func NewRegistry() *Registry {
	accounts := map[Type]*Account{
		ASSETS:      {accountType: ASSETS, name: "Assets", segment: "Assets", level: 1},
		LIABILITIES: {accountType: LIABILITIES, name: "Liabilities", segment: "Liabilities", level: 1},
		EQUITY:      {accountType: EQUITY, name: "Equity", segment: "Equity", level: 1},
		INCOME:      {accountType: INCOME, name: "Income", segment: "Income", level: 1},
		EXPENSES:    {accountType: EXPENSES, name: "Expenses", segment: "Expenses", level: 1},
	}
	index := make(map[string]*Account)
	for _, account := range accounts {
		index[account.name] = account
	}
	return &Registry{
		accounts: accounts,
		index:    index,
		parents:  make(map[*Account]*Account),
		children: make(map[*Account]set.Set[*Account]),
		swaps:    make(map[*Account]*Account),
	}
}

// Get returns an account.
func (as *Registry) Get(name string) (*Account, error) {
	as.mutex.RLock()
	res, ok := as.index[name]
	as.mutex.RUnlock()
	if ok {
		return res, nil
	}
	as.mutex.Lock()
	defer as.mutex.Unlock()
	// check if the account has been created in the meantime
	if a, ok := as.index[name]; ok {
		return a, nil
	}
	segments := strings.Split(name, ":")
	if len(segments) < 2 {
		return nil, fmt.Errorf("invalid account name: %q", name)
	}
	head, tail := segments[0], segments[1:]
	at, ok := types[head]
	if !ok {
		return nil, fmt.Errorf("account name %q has an invalid account type %q", name, segments[0])
	}
	for _, s := range tail {
		if !isValidSegment(s) {
			return nil, fmt.Errorf("account name %q has an invalid segment %q", name, s)
		}
	}
	var parent *Account
	for i := range segments {
		n := strings.Join(segments[:i+1], ":")
		parent = dict.GetDefault(as.index, n, func() *Account {
			acc := &Account{
				accountType: at,
				name:        n,
				segment:     segments[i],
				level:       i + 1,
			}
			as.parents[acc] = parent
			dict.GetDefault(as.children, parent, set.New[*Account]).Add(acc)
			return acc
		})
	}
	return parent, nil
}

func (as *Registry) Create(a *syntax.Account) (*Account, error) {
	return as.Get(a.Extract())
}

func isValidSegment(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, c := range s {
		if !(unicode.IsLetter(c) || unicode.IsDigit(c)) {
			return false
		}
	}
	return true
}

// Parent returns the parent of this account.
func (as *Registry) Parent(a *Account) *Account {
	as.mutex.RLock()
	defer as.mutex.RUnlock()
	return as.parents[a]
}

// Ancestors returns the chain of ancestors of a, including a.
func (as *Registry) Ancestors(a *Account) []*Account {
	as.mutex.RLock()
	defer as.mutex.RUnlock()
	return as.ancestors(a)
}

func (as *Registry) ancestors(a *Account) []*Account {
	var res []*Account
	if p := as.parents[a]; p != nil {
		res = as.ancestors(p)
	}
	return append(res, a)
}

// Children returns the children of this account.
func (as *Registry) Children(a *Account) []*Account {
	as.mutex.RLock()
	defer as.mutex.RUnlock()
	ch := as.children[a]
	if ch == nil {
		return nil
	}
	res := make([]*Account, 0, len(ch))
	for c := range ch {
		res = append(res, c)
	}
	return res
}

func (as *Registry) NthParent(a *Account, n int) *Account {
	as.mutex.RLock()
	defer as.mutex.RUnlock()
	if n <= 0 {
		return a
	}
	var ok bool
	for i := 0; i < n; i++ {
		a, ok = as.parents[a]
		if !ok {
			return nil
		}
	}
	return a
}

func (as *Registry) SwapType(a *Account) *Account {
	as.mutex.RLock()
	sw, ok := as.swaps[a]
	as.mutex.RUnlock()
	if ok {
		return sw
	}
	n := a.name
	switch a.Type() {
	case ASSETS:
		n = as.accounts[LIABILITIES].name + strings.TrimPrefix(n, as.accounts[ASSETS].name)
	case LIABILITIES:
		n = as.accounts[ASSETS].name + strings.TrimPrefix(n, as.accounts[LIABILITIES].name)
	case INCOME:
		n = as.accounts[EXPENSES].name + strings.TrimPrefix(n, as.accounts[INCOME].name)
	case EXPENSES:
		n = as.accounts[INCOME].name + strings.TrimPrefix(n, as.accounts[EXPENSES].name)
	}
	sw, err := as.Get(n)
	if err != nil {
		panic(err)
	}
	as.mutex.Lock()
	defer as.mutex.Unlock()
	as.swaps[a] = sw
	return sw

}
