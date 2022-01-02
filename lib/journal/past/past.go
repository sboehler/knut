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

package past

import (
	"fmt"
	"time"

	"github.com/sboehler/knut/lib/common/amounts"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
)

// PAST is a processed AST.
type PAST struct {
	Days    []*Day
	Context journal.Context
}

// MinDate returns the minimum date for this ledger, as the first
// date on which an account is opened (ignoring prices, for example).
func (l PAST) MinDate() (time.Time, bool) {
	for _, s := range l.Days {
		if len(s.AST.Openings) > 0 {
			return s.Date, true
		}
	}
	return time.Time{}, false
}

// MaxDate returns the maximum date for the given
func (l PAST) MaxDate() (time.Time, bool) {
	if len(l.Days) == 0 {
		return time.Time{}, false
	}
	return l.Days[len(l.Days)-1].Date, true
}

// Day represents a day of activity in the processed AST.
type Day struct {
	AST          *ast.Day
	Date         time.Time
	Transactions []*ast.Transaction
	Amounts      amounts.Amounts
}

// Less establishes an ordering on Day.
func (d *Day) Less(d2 *Day) bool {
	return d.Date.Before(d2.Date)
}

// Accounts keeps track of open accounts.
type Accounts map[*journal.Account]bool

// Open opens an account.
func (oa Accounts) Open(a *journal.Account) error {
	if oa[a] {
		return fmt.Errorf("account %v is already open", a)
	}
	oa[a] = true
	return nil
}

// Close closes an account.
func (oa Accounts) Close(a *journal.Account) error {
	if !oa[a] {
		return fmt.Errorf("account %v is already closed", a)
	}
	delete(oa, a)
	return nil
}

// IsOpen returns whether an account is open.
func (oa Accounts) IsOpen(a *journal.Account) bool {
	if oa[a] {
		return true
	}
	return a.Type() == journal.EQUITY
}
