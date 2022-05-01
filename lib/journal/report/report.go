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

package report

import (
	"context"
	"time"

	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/common/date"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
	"github.com/shopspring/decimal"
)

// Balance is a balance report for a range of dates.
type Balance struct {
	Dates     map[date.Period]struct{}
	Mapping   journal.Mapping
	Positions indexByAccount
}

// Subtree returns the accounts of the minimal dense subtree which
// covers the accounts in this report.
func (rep Balance) Subtree() map[*journal.Account]struct{} {
	m := make(map[*journal.Account]struct{})
	for acc := range rep.Positions {
		for p := acc; p != nil; p = p.Parent() {
			m[p] = struct{}{}
		}
	}
	return m
}

// BalanceBuilder builds a report.
type BalanceBuilder struct {
	Mapping   journal.Mapping
	Valuation bool

	Result *Balance
}

func (rb *BalanceBuilder) add(rep *Balance, b *ast.Period) {
	if rep.Positions == nil {
		rep.Positions = make(indexByAccount)
		rep.Dates = make(map[date.Period]struct{})
	}
	rep.Dates[b.Period] = struct{}{}
	a := b.Amounts
	if rb.Valuation {
		a = b.Values
	}
	for pos, val := range a {
		if val.IsZero() {
			continue
		}
		if acc := pos.Account.Map(rb.Mapping); acc != nil {
			rep.Positions.Add(acc, pos.Commodity, b.Period.End, val)
		}
	}
}

// Sink consumes the stream and produces a report.
func (rb *BalanceBuilder) Sink(ctx context.Context, inCh <-chan *ast.Period) error {
	rb.Result = new(Balance)
	for {
		d, ok, err := cpr.Pop(ctx, inCh)
		if err != nil {
			return err
		}
		if !ok {
			break
		}
		rb.add(rb.Result, d)
	}
	return nil
}

type indexByAccount map[*journal.Account]indexByCommodity

func (iba indexByAccount) Add(acc *journal.Account, com *journal.Commodity, date time.Time, val decimal.Decimal) {
	if val.IsZero() {
		return
	}
	byCommodity, ok := iba[acc]
	if !ok {
		byCommodity = make(indexByCommodity)
		iba[acc] = byCommodity
	}
	byCommodity.Add(com, date, val)
}

type indexByCommodity map[*journal.Commodity]indexByDate

func (ibc indexByCommodity) AddFrom(otherByCommodity indexByCommodity) {
	for c, otherByDate := range otherByCommodity {
		for d, v := range otherByDate {
			ibc.Add(c, d, v)
		}
	}
}

func (ibc indexByCommodity) Normalize() {
	for com, byDate := range ibc {
		if byDate.IsZero() {
			delete(ibc, com)
		}
	}
}

func (ibc indexByCommodity) Sum() map[time.Time]decimal.Decimal {
	res := make(indexByDate)
	for _, byDate := range ibc {
		res.AddFrom(byDate)
	}
	return res
}

func (ibc indexByCommodity) Add(com *journal.Commodity, date time.Time, val decimal.Decimal) {
	if val.IsZero() {
		return
	}
	byDate, ok := ibc[com]
	if !ok {
		byDate = make(indexByDate)
		ibc[com] = byDate
	}
	byDate.Add(date, val)
}

type indexByDate map[time.Time]decimal.Decimal

func (ibd indexByDate) Add(date time.Time, val decimal.Decimal) {
	if val.IsZero() {
		return
	}
	ibd[date] = ibd[date].Add(val)
}

func (ibd indexByDate) AddFrom(obd indexByDate) {
	for d, val := range obd {
		ibd.Add(d, val)
	}
}

func (ibd indexByDate) IsZero() bool {
	for _, val := range ibd {
		if !val.IsZero() {
			return false
		}
	}
	return true
}
