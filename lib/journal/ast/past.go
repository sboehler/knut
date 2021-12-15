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

package ast

import (
	"sort"
	"time"

	"github.com/sboehler/knut/lib/common/date"
	"github.com/sboehler/knut/lib/journal"
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
		if len(s.Openings) > 0 {
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

// Dates returns a series of dates which covers the first
// and last date in the ast.
func (l PAST) Dates(from, to time.Time, period date.Period) []time.Time {
	if len(l.Days) == 0 {
		return nil
	}
	var t0, t1 time.Time
	if !from.IsZero() {
		t0 = from
	} else {
		t0, _ = l.MinDate()
	}
	if !to.IsZero() {
		t1 = to
	} else {
		t1, _ = l.MaxDate()
	}
	return date.Series(t0, t1, period)
}

// ActualDates returns a series like Dates, but containing the latest available,
// actual dates from the days in the ast. That is, an element of the result
// array is either the zero date (if it is before the first date in the ledger),
// or the latest date in the ledger which is smaller or equal than the corresponding
// element in the input array.
func (l PAST) ActualDates(ds []time.Time) []time.Time {
	var actuals = make([]time.Time, 0, len(ds))
	for _, date := range ds {
		if len(l.Days) == 0 || date.Before(l.Days[0].Date) {
			// no days in the ledger, or date before all ledger days
			actuals = append(actuals, time.Time{})
			continue
		}
		index := sort.Search(len(l.Days), func(i int) bool { return !l.Days[i].Date.Before(date) })
		if index == len(l.Days) {
			// all days are after the date, use the latest one
			actuals = append(actuals, l.Days[len(l.Days)-1].Date)
			continue
		}
		actuals = append(actuals, l.Days[index].Date)
	}
	return actuals
}
