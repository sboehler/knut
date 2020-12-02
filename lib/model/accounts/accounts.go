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
)

// AccountType is the type of an account.
type AccountType int

const (
	// TBD is an account which has not yet been determined.
	TBD AccountType = iota
	// ASSETS represents an asset account.
	ASSETS
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
	case TBD:
		return "TBD"
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
var AccountTypes = []AccountType{ASSETS, LIABILITIES, EQUITY, INCOME, EXPENSES, TBD}

var accountTypes = map[string]AccountType{
	"TBD":         TBD,
	"Assets":      ASSETS,
	"Liabilities": LIABILITIES,
	"Equity":      EQUITY,
	"Expenses":    EXPENSES,
	"Income":      INCOME,
}

var (
	mutex     = sync.RWMutex{}
	accounts  = make(map[string]*Account)
	maxLength = 0
)

func get(name string) (*Account, bool) {
	mutex.RLock()
	defer mutex.RUnlock()
	c, ok := accounts[name]
	return c, ok
}

func create(name string) (*Account, error) {
	mutex.Lock()
	defer mutex.Unlock()
	if a, ok := accounts[name]; ok {
		return a, nil
	}
	if t, ok := accountTypes[strings.SplitN(name, ":", 2)[0]]; ok {
		a := &Account{
			accountType: t,
			name:        name,
		}
		accounts[name] = a
		if maxLength < len(name) {
			maxLength = len(name)
		}
		return a, nil
	}
	return nil, fmt.Errorf("invalid account name: %q", name)
}

var valuationAccount, retainedEarningsAccount, tbdAccount *Account

func init() {
	var err error
	valuationAccount, err = Get("Equity:Valuation")
	if err != nil {
		panic("Could not create valuationAccount")
	}
	retainedEarningsAccount, err = Get("Equity:RetainedEarnings")
	if err != nil {
		panic("Could not create valuationAccount")
	}
	tbdAccount, err = Get("TBD")
	if err != nil {
		panic("Could not create TBD account")
	}
}

// ValuationAccount returns the account for automatic valuation bookings.
func ValuationAccount() *Account {
	return valuationAccount
}

// RetainedEarningsAccount returns the account for automatic valuation bookings.
func RetainedEarningsAccount() *Account {
	return retainedEarningsAccount
}

// TBDAccount returns the TBD account.
func TBDAccount() *Account {
	return tbdAccount
}

// Account represents an account which can be used in bookings.
type Account struct {
	accountType AccountType
	name        string
}

// Get creates a new account.
func Get(name string) (*Account, error) {
	if a, ok := get(name); ok {
		return a, nil
	}
	return create(name)
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

// RightPad returns a string with the account name, padded to the right.
func (a Account) RightPad() string {
	b := strings.Builder{}
	b.WriteString(a.name)
	for i := len(a.name); i < maxLength; i++ {
		b.WriteRune(' ')
	}
	return b.String()
}

func (a Account) String() string {
	return a.name
}
