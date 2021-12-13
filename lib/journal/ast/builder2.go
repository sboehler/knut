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
	"sort"
	"time"

	"github.com/sboehler/knut/lib/journal"
)

// FromDirectives2 reads directives from the given channel and
// builds a Ledger if successful.
func FromDirectives2(ctx journal.Context, filter journal.Filter, results <-chan interface{}) (*AST, error) {
	var b = NewBuilder2(ctx)
	for res := range results {
		switch t := res.(type) {
		case error:
			return nil, t
		case *Open:
			var s = b.getOrCreate(t.Date)
			s.Openings = append(s.Openings, t)
		case *Price:
			var s = b.getOrCreate(t.Date)
			s.Prices = append(s.Prices, t)
		case *Transaction:
			var s = b.getOrCreate(t.Date)
			s.Transactions = append(s.Transactions, t)
		case *Assertion:
			var s = b.getOrCreate(t.Date)
			s.Assertions = append(s.Assertions, t)
		case *Value:
			var s = b.getOrCreate(t.Date)
			s.Values = append(s.Values, t)
		case *Close:
			var s = b.getOrCreate(t.Date)
			s.Closings = append(s.Closings, t)
		default:
			return nil, fmt.Errorf("unknown: %#v", t)
		}
	}
	return b.Build(), nil
}

// Builder2 maps dates to days
type Builder2 struct {
	days    map[time.Time]*Day
	Context journal.Context
}

// NewBuilder2 creates a new builder2.
func NewBuilder2(ctx journal.Context) *Builder2 {
	return &Builder2{make(map[time.Time]*Day), ctx}
}

// Build creates a new
func (b *Builder2) Build() *AST {
	var result = make([]*Day, 0, len(b.days))
	for _, s := range b.days {
		result = append(result, s)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Date.Before(result[j].Date)
	})
	return &AST{
		Days:    result,
		Context: b.Context,
	}

}

func (b *Builder2) getOrCreate(d time.Time) *Day {
	s, ok := b.days[d]
	if !ok {
		s = &Day{Date: d}
		b.days[d] = s
	}
	return s
}
