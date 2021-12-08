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
	"time"

	"github.com/sboehler/knut/lib/date"
)

// Day groups all commands for a given date.
type Day struct {
	Date         time.Time
	Prices       []Price
	Assertions   []Assertion
	Values       []Value
	Openings     []Open
	Transactions []Transaction
	Closings     []Close
}

// Ledger is a
type Ledger struct {
	Days    []*Day
	Context Context
}

// MinDate returns the minimum date for this ledger, as the first
// date on which an account is opened (ignoring prices, for example).
func (l Ledger) MinDate() (time.Time, bool) {
	for _, s := range l.Days {
		if len(s.Openings) > 0 {
			return s.Date, true
		}
	}
	return time.Time{}, false
}

// MaxDate returns the maximum date for the given
func (l Ledger) MaxDate() (time.Time, bool) {
	if len(l.Days) == 0 {
		return time.Time{}, false
	}
	return l.Days[len(l.Days)-1].Date, true
}

// Dates returns a series of dates which covers the first
// and last date in the ledger.
func (l Ledger) Dates(from, to *time.Time, period date.Period) []time.Time {
	if len(l.Days) == 0 {
		return nil
	}
	var t0, t1 time.Time
	if from != nil {
		t0 = *from
	} else {
		t0, _ = l.MinDate()
	}
	if to != nil {
		t1 = *to
	} else {
		t1, _ = l.MaxDate()
	}
	return date.Series(t0, t1, period)
}
