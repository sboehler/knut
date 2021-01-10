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

package ledger

import (
	"time"

	"github.com/sboehler/knut/lib/model"
)

// Step groups all commands for a given date.
type Step struct {
	Date         time.Time
	Prices       []*model.Price
	Assertions   []*model.Assertion
	Values       []*model.Value
	Openings     []*model.Open
	Transactions []*model.Transaction
	Closings     []*model.Close
}

// Ledger is a ledger.
type Ledger []*Step

// MinDate returns the minimum date for this ledger, as the first
// date on which an account is opened (ignoring prices, for example).
func (l Ledger) MinDate() (time.Time, bool) {
	for _, s := range l {
		if len(s.Openings) > 0 {
			return s.Date, true
		}
	}
	return time.Time{}, false
}

// MaxDate returns the maximum date for the given ledger.
func (l Ledger) MaxDate() (time.Time, bool) {
	if len(l) == 0 {
		return time.Time{}, false
	}
	return l[len(l)-1].Date, true
}
