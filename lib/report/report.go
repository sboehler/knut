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
	"time"

	"github.com/sboehler/knut/lib/balance"
	"github.com/sboehler/knut/lib/ledger"
	"github.com/shopspring/decimal"
)

// Report is a balance report for a range of dates.
type Report struct {
	Dates     []time.Time
	Value     bool
	Mapping   ledger.Mapping
	Positions indexByAccount
}

// Add adds a balance to this report.
func (rep *Report) Add(b *balance.Balance) {
	rep.Dates = append(rep.Dates, b.Date)
	if rep.Positions == nil {
		rep.Positions = make(indexByAccount)
	}
	var bp = b.Amounts
	if rep.Value {
		bp = b.Values
	}
	for pos, val := range bp {
		if val.IsZero() {
			continue
		}
		if acc := pos.Account.Map(rep.Mapping); acc != nil {
			rep.Positions.Add(acc, pos.Commodity, b.Date, val)
		}
	}
}

// Subtree returns the accounts of the minimal dense subtree which
// covers the accounts in this report.
func (rep Report) Subtree() map[*ledger.Account]struct{} {
	m := make(map[*ledger.Account]struct{})
	for acc := range rep.Positions {
		for p := acc; p != nil; p = p.Parent() {
			m[p] = struct{}{}
		}
	}
	return m
}

type indexByAccount map[*ledger.Account]indexByCommodity

func (iba indexByAccount) Add(acc *ledger.Account, com *ledger.Commodity, date time.Time, val decimal.Decimal) {
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

type indexByCommodity map[*ledger.Commodity]indexByDate

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

func (ibc indexByCommodity) Add(com *ledger.Commodity, date time.Time, val decimal.Decimal) {
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
