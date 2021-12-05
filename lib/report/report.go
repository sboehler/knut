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
	"fmt"
	"time"

	"github.com/sboehler/knut/lib/balance"
	"github.com/sboehler/knut/lib/ledger"
	"github.com/shopspring/decimal"
)

// Report is a balance report for a range of dates.
type Report struct {
	Context   ledger.Context
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
		acc := pos.Account.Map(rep.Mapping)
		fmt.Println(acc)
		if acc == nil {
			continue
		}
		rep.Positions.Add(acc, pos.Commodity, b.Date, val)
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
	ibc, ok := iba[acc]
	if !ok {
		ibc = make(indexByCommodity)
		iba[acc] = ibc
	}
	ibc.Add(com, date, val)
}

type indexByCommodity map[*ledger.Commodity]indexByDate

func (ibc indexByCommodity) AddOther(obc indexByCommodity) {
	for c, obd := range obc {
		for d, v := range obd {
			ibd, ok := ibc[c]
			if !ok {
				ibd = make(indexByDate)
				ibc[c] = ibd
			}
			ibd[d] = ibd[d].Add(v)
		}
	}
}

func (ibc indexByCommodity) Add(com *ledger.Commodity, date time.Time, val decimal.Decimal) {
	ibd, ok := ibc[com]
	if !ok {
		ibd = make(indexByDate)
		ibc[com] = ibd
	}
	ibd.Add(date, val)
}

type indexByDate map[time.Time]decimal.Decimal

func (ibd indexByDate) Add(date time.Time, val decimal.Decimal) {
	ibd[date] = ibd[date].Add(val)
}
