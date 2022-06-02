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
	"sort"
	"time"

	"github.com/sboehler/knut/lib/common/amounts2"
	"github.com/sboehler/knut/lib/common/table"
	"github.com/sboehler/knut/lib/journal"
	"github.com/shopspring/decimal"
)

// Renderer2 renders a report.
type Renderer2 struct {
	Context            journal.Context
	ShowCommodities    bool
	SortAlphabetically bool
	Valuation          *journal.Commodity

	amounts amounts2.Amounts
	table   *table.Table
	dates   []time.Time
}

// Render renders a report.
func (rn *Renderer2) Render(r amounts2.Amounts) *table.Table {
	rn.amounts = r

	// sort dates
	var (
		dates       = make(map[time.Time]struct{})
		accounts    = make(map[*journal.Account]struct{})
		commodities = make(map[*journal.Commodity]struct{})
	)
	for k := range rn.amounts {
		if !k.Date.IsZero() {
			dates[k.Date] = struct{}{}
		}
		accounts[k.Account] = struct{}{}
		commodities[k.Commodity] = struct{}{}
	}
	for d := range dates {
		rn.dates = append(rn.dates, d)
	}
	sort.Slice(rn.dates, func(i, j int) bool { return rn.dates[i].Before(rn.dates[j]) })

	rn.table = table.New(1, len(rn.dates))
	rn.table.AddSeparatorRow()

	header := rn.table.AddRow().AddText("Account", table.Center)
	for _, d := range rn.dates {
		header.AddText(d.Format("2006-01-02"), table.Center)
	}
	rn.table.AddSeparatorRow()

	subtree := rn.subtree(accounts)
	var al, eie []*journal.Account
	for _, acc := range rn.Context.Accounts().PreOrder() {
		if _, ok := subtree[acc]; !ok {
			continue
		}
		if acc.IsAL() {
			al = append(al, acc)
		} else {
			eie = append(eie, acc)
		}
	}
	sides := []struct {
		neg    bool
		accs   []*journal.Account
		totals map[time.Time]decimal.Decimal
	}{
		{
			neg:    false,
			accs:   al,
			totals: make(map[time.Time]decimal.Decimal),
		},
		{
			neg:    true,
			accs:   eie,
			totals: make(map[time.Time]decimal.Decimal),
		},
	}
	for _, side := range sides {
		for i, a := range side.accs {
			if i > 0 && a.Level() == 1 {
				rn.table.AddEmptyRow()
			}
			row := rn.table.AddRow().AddIndented(a.Segment(), 2*(a.Level()-1))
			line := make(map[time.Time]decimal.Decimal)
			for _, d := range rn.dates {
				for c := range rn.Context.Commodities().Enumerate() {
					k := amounts2.Key{Date: d, Account: a, Commodity: c, Valuation: rn.Valuation}
					if v, ok := rn.amounts[k]; ok {
						line[d] = line[d].Add(v)
					}
				}
			}
			for _, d := range rn.dates {
				if v, ok := line[d]; ok && !v.IsZero() {
					if side.neg {
						row.AddNumber(v.Neg())
						side.totals[d] = side.totals[d].Add(v.Neg())
					} else {
						row.AddNumber(v)
						side.totals[d] = side.totals[d].Add(v)
					}
				} else {
					row.AddEmpty()
				}
			}
		}
		rn.table.AddEmptyRow()
		row := rn.table.AddRow().AddText("Total", table.Left)
		for _, d := range rn.dates {
			row.AddNumber(side.totals[d])
		}
		rn.table.AddSeparatorRow()
	}
	row := rn.table.AddRow().AddText("Delta", table.Left)
	for _, d := range rn.dates {
		row.AddNumber(sides[0].totals[d].Sub(sides[1].totals[d]))
	}
	rn.table.AddSeparatorRow()
	return rn.table
}

func (rn Renderer2) subtree(as map[*journal.Account]struct{}) map[*journal.Account]struct{} {
	m := make(map[*journal.Account]struct{})
	for acc := range as {
		for p := acc; p != nil; p = p.Parent() {
			m[p] = struct{}{}
		}
	}
	return m
}
