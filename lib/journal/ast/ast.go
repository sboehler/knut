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
	"fmt"
	"time"

	"github.com/sboehler/knut/lib/journal"
)

// AST1 maps dates to days
type AST1 struct {
	Days    map[time.Time]*Day
	Context journal.Context
}

// Day returns the Day for the given date.
func (b *AST1) Day(d time.Time) *Day {
	s, ok := b.Days[d]
	if !ok {
		s = &Day{Date: d}
		b.Days[d] = s
	}
	return s
}

// Day groups all commands for a given date.
type Day struct {
	Date         time.Time
	Prices       []*Price
	Assertions   []*Assertion
	Values       []*Value
	Openings     []*Open
	Transactions []*Transaction
	Closings     []*Close
}

// Less establishes an ordering on Day.
func (d *Day) Less(d2 *Day) bool {
	return d.Date.Before(d2.Date)
}

// FromDirectives2 reads directives from the given channel and
// builds a Ledger if successful.
func FromDirectives2(ctx journal.Context, results <-chan interface{}) (*AST1, error) {
	var b = &AST1{
		Context: ctx,
		Days:    make(map[time.Time]*Day),
	}
	for res := range results {
		switch t := res.(type) {
		case error:
			return nil, t
		case *Open:
			var s = b.Day(t.Date)
			s.Openings = append(s.Openings, t)
		case *Price:
			var s = b.Day(t.Date)
			s.Prices = append(s.Prices, t)
		case *Transaction:
			var s = b.Day(t.Date)
			s.Transactions = append(s.Transactions, t)
		case *Assertion:
			var s = b.Day(t.Date)
			s.Assertions = append(s.Assertions, t)
		case *Value:
			var s = b.Day(t.Date)
			s.Values = append(s.Values, t)
		case *Close:
			var s = b.Day(t.Date)
			s.Closings = append(s.Closings, t)
		default:
			return nil, fmt.Errorf("unknown: %#v", t)
		}
	}
	return b, nil
}
