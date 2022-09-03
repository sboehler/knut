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
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"sync"
	"unicode"

	"github.com/sboehler/knut/lib/common/compare"
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

func CompareAccountTypes(t1, t2 AccountType) compare.Order {
	if t1 == t2 {
		return compare.Equal
	}
	if t1 < t2 {
		return compare.Smaller
	}
	return compare.Greater
}

// Account represents an account which can be used in bookings.
type Account struct {
	accountType AccountType
	name        string
	segment     string
	level       int
}

// Split returns the account name split into segments.
func (a *Account) Split() []string {
	return strings.Split(a.name, ":")
}

// Name returns the name of this account.
func (a Account) Name() string {
	return a.name
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

func (a *Account) Less(a2 *Account) bool {
	if a.accountType != a2.accountType {
		return a.accountType < a2.accountType
	}
	return a.name < a2.name
}

// IsAL returns whether this account is an asset or liability account.
func (a Account) IsAL() bool {
	return a.accountType == ASSETS || a.accountType == LIABILITIES
}

// WriteTo writes the account to the writer.
func (a Account) WriteTo(w io.Writer) (int64, error) {
	n, err := fmt.Fprint(w, a.Name())
	return int64(n), err
}

func (a Account) String() string {
	return a.name
}

func Compare(a1, a2 *Account) compare.Order {
	o := CompareAccountTypes(a1.accountType, a2.accountType)
	if o != compare.Equal {
		return o
	}
	return compare.Ordered(a1.name, a2.name)
}

func CompareWeighted(jctx Context, w map[*Account]float64) compare.Compare[*Account] {
	// compareSameType compares two accounts which are known to have the same account type.
	var compareSameType func(a1, a2 *Account) compare.Order
	compareSameType = func(a1, a2 *Account) compare.Order {
		p1, p2 := jctx.Accounts().Parent(a1), jctx.Accounts().Parent(a2)
		if p1 == nil {
			if p2 == nil {
				return compare.Equal
			}
			return compare.Smaller
		}
		// p1 != nil
		if p2 == nil {
			return compare.Greater
		}
		// p1 != nil && p2 != nil
		if p1 == p2 {
			if o := compare.Ordered(w[a1], w[a2]); o != compare.Equal {
				return o
			}
			return compare.Ordered(a1.Name(), a2.Name())
		}
		// recurse until one is nil or both are identical
		return compareSameType(p1, p2)
	}

	return func(a1, a2 *Account) compare.Order {
		if o := CompareAccountTypes(a1.accountType, a2.accountType); o != compare.Equal {
			// weights don't influence order of account types
			return o
		}
		return compareSameType(a1, a2)
	}
}

// Accounts is a thread-safe collection of accounts.
type Accounts struct {
	mutex    sync.RWMutex
	index    map[string]*Account
	accounts map[AccountType]*Account
	children map[*Account]map[*Account]bool
	parents  map[*Account]*Account
}

// NewAccounts creates a new thread-safe collection of accounts.
func NewAccounts() *Accounts {
	accounts := map[AccountType]*Account{
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
	return &Accounts{
		accounts: accounts,
		index:    index,
		parents:  make(map[*Account]*Account),
		children: make(map[*Account]map[*Account]bool),
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
	segments := strings.Split(name, ":")
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
	var parent *Account
	for i := range segments {
		n := strings.Join(segments[:i+1], ":")
		acc, ok := as.index[n]
		if !ok {
			acc = &Account{
				accountType: at,
				name:        n,
				segment:     segments[i],
				level:       i + 1,
			}
			as.index[n] = acc
			as.parents[acc] = parent
			ch, ok := as.children[parent]
			if !ok {
				ch = make(map[*Account]bool)
				as.children[parent] = ch
			}
			ch[acc] = true
		}
		parent = acc
	}
	return parent, nil
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

// PreOrder iterates over accounts in post-compare.
func (as *Accounts) PreOrder() []*Account {
	as.mutex.RLock()
	defer as.mutex.RUnlock()
	var res []*Account
	for _, at := range AccountTypes {
		res = as.pre(as.accounts[at], res)
	}
	return res
}

// PostOrder iterates over accounts in post-compare.
func (as *Accounts) PostOrder() []*Account {
	as.mutex.RLock()
	defer as.mutex.RUnlock()
	var res []*Account
	for _, at := range AccountTypes {
		res = as.post(as.accounts[at], res)
	}
	return res
}

// SortedPreOrder sorts every level according to the weights supplied. The
// given weights must be dense.
func (as *Accounts) SortedPreOrder(weights map[*Account]float64) []*Account {
	as.mutex.RLock()
	defer as.mutex.RUnlock()
	var res []*Account
	for _, at := range AccountTypes {
		res = as.sorted(as.accounts[at], res, weights)
	}
	return res
}

// Parent returns the parent of this account.
func (as *Accounts) Parent(a *Account) *Account {
	as.mutex.RLock()
	defer as.mutex.RUnlock()
	return as.parents[a]
}

// Children returns the children of this account.
func (as *Accounts) Children(a *Account) []*Account {
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

func (as *Accounts) Densify(m map[*Account]struct{}) map[*Account]struct{} {
	res := make(map[*Account]struct{})
	for a := range m {
		for ; a != nil; a = as.parents[a] {
			res[a] = struct{}{}
		}
	}
	return res
}

// Descendents returns all the descendents of this account, not including
// the account itself.
func (as *Accounts) Descendents(a *Account) []*Account {
	as.mutex.RLock()
	defer as.mutex.RUnlock()
	return as.descendents(a, nil)
}

// Descendents returns all the descendents of this account, not including
// the account itself.
func (as *Accounts) descendents(a *Account, res []*Account) []*Account {
	for ch := range as.children[a] {
		res = append(res, ch)
		res = as.descendents(ch, res)
	}
	return res
}

// Map maps an account to itself or to one of its ancestors.
func (as *Accounts) Map(a *Account, m Mapping) *Account {
	as.mutex.RLock()
	defer as.mutex.RUnlock()
	if len(m) == 0 {
		return a
	}
	level := m.level(a)
	if level >= a.level {
		return a
	}
	return as.nthParent(a, a.level-level)
}

func (as *Accounts) nthParent(a *Account, n int) *Account {
	if n <= 0 {
		return a
	}
	p, ok := as.parents[a]
	if !ok {
		return nil
	}
	return as.nthParent(p, n-1)
}

func (as *Accounts) pre(a *Account, acc []*Account) []*Account {
	acc = append(acc, a)
	for c := range as.children[a] {
		acc = as.pre(c, acc)
	}
	return acc
}

func (as *Accounts) post(a *Account, acc []*Account) []*Account {
	for c := range as.children[a] {
		acc = as.post(c, acc)
	}
	acc = append(acc, a)
	return acc
}

func (as *Accounts) sorted(a *Account, in []*Account, weights map[*Account]float64) []*Account {
	childMap := as.children[a]
	children := make([]*Account, 0, len(childMap))
	for ch := range childMap {
		children = append(children, ch)
	}
	sort.Slice(children, func(i int, j int) bool {
		return weights[children[i]] > weights[children[j]]
	})
	in = append(in, a)
	for _, ch := range children {
		in = as.sorted(ch, in, weights)
	}
	return in
}

// Rule is a rule to shorten accounts which match the given regex.
type Rule struct {
	Level int
	Regex *regexp.Regexp
}

func (r Rule) String() string {
	return fmt.Sprintf("%d,%v", r.Level, r.Regex)
}

// Mapping is a set of mapping rules.
type Mapping []Rule

func (m Mapping) String() string {
	var s []string
	for _, r := range m {
		s = append(s, r.String())
	}
	return strings.Join(s, ", ")
}

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

func MapAccount(m Mapping) func(Context, *Account) *Account {
	return func(jctx Context, a *Account) *Account {
		return jctx.Accounts().Map(a, m)
	}
}
