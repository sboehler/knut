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
	"io"
	"regexp"
	"sort"
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
	index    map[string]*Account
	accounts map[AccountType]*Account
}

// NewAccounts creates a new thread-safe collection of accounts.
func NewAccounts() *Accounts {
	return &Accounts{
		index: make(map[string]*Account),
		accounts: map[AccountType]*Account{
			ASSETS:      {accountType: ASSETS, segment: "Assets", level: 1},
			LIABILITIES: {accountType: LIABILITIES, segment: "Liabilities", level: 1},
			EQUITY:      {accountType: EQUITY, segment: "Equity", level: 1},
			INCOME:      {accountType: INCOME, segment: "Income", level: 1},
			EXPENSES:    {accountType: EXPENSES, segment: "Expenses", level: 1},
		},
	}
}

// Get returns an account.
func (as *Accounts) Get(name string) (*Account, error) {
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
	var segments = strings.Split(name, ":")
	if len(segments) < 2 {
		return nil, fmt.Errorf("invalid account name: %q", name)
	}
	head, tail := segments[0], segments[1:]
	at, ok := accountTypes[head]
	if !ok {
		return nil, fmt.Errorf("account name %q has an invalid account type %q", name, segments[0])
	}
	for _, s := range tail {
		if !isValidSegment(s) {
			return nil, fmt.Errorf("account name %q has an invalid segment %q", name, s)
		}
	}
	root := as.accounts[at]
	res = root.insert(tail)
	as.index[name] = res
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

// PreOrder iterates over accounts in post-order.
func (as *Accounts) PreOrder() <-chan *Account {
	as.mutex.RLock()
	defer as.mutex.RUnlock()
	ch := make(chan *Account)
	go func() {
		defer close(ch)
		for _, at := range AccountTypes {
			as.accounts[at].pre(ch)
		}
	}()
	return ch
}

// PostOrder iterates over accounts in post-order.
func (as *Accounts) PostOrder() <-chan *Account {
	as.mutex.RLock()
	defer as.mutex.RUnlock()
	ch := make(chan *Account)
	go func() {
		defer close(ch)
		for _, root := range as.accounts {
			root.post(ch)
		}
	}()
	return ch
}

// Account represents an account which can be used in bookings.
type Account struct {
	accountType AccountType
	segment     string
	parent      *Account
	children    []*Account
	level       int
}

func (a *Account) insert(segments []string) *Account {
	if len(segments) == 0 {
		return a
	}
	head, tail := segments[0], segments[1:]
	index := sort.Search(len(a.children), func(i int) bool { return a.children[i].segment >= head })
	if index == len(a.children) || a.children[index].segment != head {
		a.children = append(a.children, nil)
		copy(a.children[index+1:], a.children[index:])
		a.children[index] = &Account{
			segment:     head,
			accountType: a.accountType,
			parent:      a,
			level:       a.level + 1,
		}
	}
	return a.children[index].insert(tail)
}

// Split returns the account name split into segments.
func (a *Account) Split() []string {
	if a == nil {
		return nil
	}
	var res []string
	if a.parent != nil {
		res = a.parent.Split()
	}
	return append(res, a.segment)
}

// Name returns the name of this account.
func (a Account) Name() string {
	return strings.Join(a.Split(), ":")
}

// Segment returns the name of this account.
func (a Account) Segment() string {
	return a.segment
}

// Level returns the name of this account.
func (a Account) Level() int {
	return a.level
}

// Type returns the account type.
func (a Account) Type() AccountType {
	return a.accountType
}

// Parent returns the parent of this account.
func (a Account) Parent() *Account {
	return a.parent
}

// Descendents returns all the descendents of this account, not including
// the account itself.
func (a Account) Descendents() []*Account {
	var res []*Account
	for _, ch := range a.children {
		res = append(res, ch.Descendents()...)
	}
	return res
}

// WriteTo writes the account to the writer.
func (a Account) WriteTo(w io.Writer) (int64, error) {
	n, err := fmt.Fprint(w, a.Name())
	return int64(n), err
}

func (a Account) String() string {
	return a.Name()
}

// Map maps an account to itself or to one of its ancestors.
func (a *Account) Map(m Mapping) *Account {
	if len(m) == 0 {
		return a
	}
	level := m.level(a)
	if level >= a.level {
		return a
	}
	return a.nthParent(a.level - level)
}

func (a *Account) nthParent(n int) *Account {
	if n <= 0 {
		return a
	}
	if a.parent == nil {
		return nil
	}
	return a.parent.nthParent(n - 1)
}

func (a *Account) pre(ch chan<- *Account) {
	ch <- a
	for _, c := range a.children {
		c.pre(ch)
	}
}

func (a *Account) post(ch chan<- *Account) {
	for _, c := range a.children {
		c.post(ch)
	}
	ch <- a
}

// Rule is a rule to shorten accounts which match the given regex.
type Rule struct {
	Level int
	Regex *regexp.Regexp
}

// Mapping is a set of mapping rules.
type Mapping []Rule

// level returns the level to which an account should be shortened.
func (m Mapping) level(a *Account) int {
	var (
		name  = a.Name()
		level = a.level
	)
	for _, c := range m {
		if (c.Regex == nil || c.Regex.MatchString(name)) && c.Level < level {
			level = c.Level
		}
	}
	return level
}
