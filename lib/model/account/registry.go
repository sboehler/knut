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

	"github.com/sboehler/knut/lib/common/multimap"
	"github.com/sboehler/knut/lib/syntax"
)

// Registry is a thread-safe collection of accounts.
type Registry struct {
	mutex    sync.RWMutex
	index    map[string]*Account
	accounts *multimap.Node[*Account]
	swaps    map[*Account]*Account
}

// NewRegistry creates a new thread-safe collection of accounts.
func NewRegistry() *Registry {
	reg := &Registry{
		accounts: multimap.New[*Account](""),
		index:    make(map[string]*Account),
		swaps:    make(map[*Account]*Account),
	}
	for _, t := range types {
		reg.Get(t.String())
	}

	return reg
}

// Get returns an account.
func (as *Registry) Get(name string) (*Account, error) {
	as.mutex.RLock()
	res, ok := as.index[name]
	as.mutex.RUnlock()
	if ok {
		return res, nil
	}
	return as.getOrCreatePath(strings.Split(name, ":"))
}

// Get returns an account.
func (as *Registry) GetPath(segments []string) (*Account, error) {
	as.mutex.RLock()
	res, ok := as.accounts.GetPath(segments)
	as.mutex.RUnlock()
	if ok {
		return res.Value, nil
	}
	return as.getOrCreatePath(segments)
}

func (as *Registry) getOrCreatePath(segments []string) (*Account, error) {
	as.mutex.Lock()
	defer as.mutex.Unlock()
	if res, ok := as.accounts.GetPath(segments); ok {
		return res.Value, nil
	}
	if len(segments) == 0 {
		return nil, fmt.Errorf("invalid account: %s", segments)
	}
	head, tail := segments[0], segments[1:]
	accountType, ok := types[head]
	if !ok {
		return nil, fmt.Errorf("account %s has an invalid account type %s", segments, head)
	}
	for _, s := range tail {
		if !isValidSegment(s) {
			return nil, fmt.Errorf("account  %s has an invalid segment %q", segments, s)
		}
	}
	current := as.accounts
	for i, segment := range segments {
		if ch, ok := current.Get(segment); ok {
			current = ch
			continue
		}
		var err error
		if current, err = current.Create(segment); err != nil {
			return nil, err
		}
		name := strings.Join(segments[:i+1], ":")
		current.Value = &Account{
			accountType: accountType,
			name:        name,
			segments:    strings.Split(name, ":"),
		}
		as.index[name] = current.Value
	}
	return current.Value, nil
}

func (as *Registry) MustGet(name string) *Account {
	a, err := as.Get(name)
	if err != nil {
		panic(err)
	}
	return a
}

func (as *Registry) MustGetPath(ss []string) *Account {
	res, err := as.GetPath(ss)
	if err != nil {
		panic(fmt.Sprintf("account %s not found: %v", ss, err))
	}
	return res
}

func (as *Registry) Create(a syntax.Account) (*Account, error) {
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
		n = LIABILITIES.String() + strings.TrimPrefix(n, ASSETS.String())
	case LIABILITIES:
		n = ASSETS.String() + strings.TrimPrefix(n, LIABILITIES.String())
	case INCOME:
		n = EXPENSES.String() + strings.TrimPrefix(n, INCOME.String())
	case EXPENSES:
		n = INCOME.String() + strings.TrimPrefix(n, EXPENSES.String())
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

// TBDAccount returns the TBD account.
func (as *Registry) TBDAccount() *Account {
	return as.MustGet("Expenses:TBD")
}

// ValuationAccountFor returns the valuation account which corresponds to
// the given Asset or Liability account.
func (as *Registry) ValuationAccountFor(a *Account) *Account {
	segments := append(as.MustGet("Income").Segments(), a.Segments()[1:]...)
	return as.MustGet(strings.Join(segments, ":"))
}
