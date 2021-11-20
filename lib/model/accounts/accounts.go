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

package accounts

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"unicode"
)

// AccountType is the type of an account.
type AccountType int

const (
	// ASSETS represents an asset account.
	ASSETS AccountType = iota
	// LIABILITIES represents a liability account.
	LIABILITIES
	// EQUITY represents an equity account.
	EQUITY
	// INCOME represents an income account.
	INCOME
	// EXPENSES represents an expenses account.
	EXPENSES
)

func (t AccountType) String() string {
	switch t {
	case ASSETS:
		return "Assets"
	case LIABILITIES:
		return "Liabilities"
	case EQUITY:
		return "Equity"
	case INCOME:
		return "Income"
	case EXPENSES:
		return "Expenses"
	}
	return ""
}

// AccountTypes is an array with the ordered accont types.
var AccountTypes = []AccountType{ASSETS, LIABILITIES, EQUITY, INCOME, EXPENSES}

var accountTypes = map[string]AccountType{
	"Assets":      ASSETS,
	"Liabilities": LIABILITIES,
	"Equity":      EQUITY,
	"Expenses":    EXPENSES,
	"Income":      INCOME,
}

// Accounts is a thread-safe collection of accounts.
type Accounts struct {
	mutex    sync.RWMutex
	accounts map[string]*Account
}

// New creates a new thread-safe collection of accounts.
func New() *Accounts {
	return &Accounts{
		accounts: make(map[string]*Account),
	}
}

func (a *Accounts) Get(name string) (*Account, error) {
	a.mutex.RLock()
	res, ok := a.accounts[name]
	a.mutex.RUnlock()
	if !ok {
		a.mutex.Lock()
		defer a.mutex.Unlock()
		// check if the account has been created in the meantime
		if a, ok := a.accounts[name]; ok {
			return a, nil
		}
		var segments = strings.Split(name, ":")
		if len(segments) < 2 {
			return nil, fmt.Errorf("invalid account name: %q", name)
		}
		at, ok := accountTypes[segments[0]]
		if !ok {
			return nil, fmt.Errorf("account name %q has an invalid account type %q", name, segments[0])
		}
		for _, s := range segments[1:] {
			if !isValidSegment(s) {
				return nil, fmt.Errorf("account name %q has an invalid segment %q", name, s)
			}
		}
		res = &Account{
			accountType: at,
			name:        name,
		}
		a.accounts[name] = res
	}
	return res, nil
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

// Account represents an account which can be used in bookings.
type Account struct {
	accountType AccountType
	name        string
}

// Split returns the account name split into segments.
func (a Account) Split() []string {
	return strings.Split(a.name, ":")
}

// Type returns the account type.
func (a Account) Type() AccountType {
	return a.accountType
}

// WriteTo writes the account to the writer.
func (a Account) WriteTo(w io.Writer) (int64, error) {
	n, err := fmt.Fprint(w, a.name)
	return int64(n), err
}

func (a Account) String() string {
	return a.name
}
