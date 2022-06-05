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
		totals amounts2.Amounts
	}{
		{
			neg:    false,
			accs:   al,
			totals: make(amounts2.Amounts),
		},
		{
			neg:    true,
			accs:   eie,
			totals: make(amounts2.Amounts),
		},
	}
	for _, side := range sides {
		for i, a := range side.accs {
			if i > 0 && a.Level() == 1 {
				rn.table.AddEmptyRow()
			}
			row := rn.table.AddRow().AddIndented(a.Segment(), 2*(a.Level()-1))
			line := make(amounts2.Amounts)
			for _, d := range rn.dates {
				for c := range rn.Context.Commodities().Enumerate() {
					k := amounts2.Key{Date: d, Account: a, Commodity: c, Valuation: rn.Valuation}
					if v, ok := rn.amounts[k]; ok {
						dk := amounts2.Key{Date: d}
						line[dk] = line[dk].Add(v)
					}
				}
			}
			for _, d := range rn.dates {
				k := amounts2.Key{Date: d}
				if v, ok := line[k]; ok && !v.IsZero() {
					if side.neg {
						row.AddNumber(v.Neg())
						side.totals[k] = side.totals[k].Add(v.Neg())
					} else {
						row.AddNumber(v)
						side.totals[k] = side.totals[k].Add(v)
					}
				} else {
					row.AddEmpty()
				}
			}
		}
		rn.table.AddEmptyRow()
		row := rn.table.AddRow().AddText("Total", table.Left)
		for _, d := range rn.dates {
			row.AddNumber(side.totals[amounts2.Key{Date: d}])
		}
		rn.table.AddSeparatorRow()
	}
	row := rn.table.AddRow().AddText("Delta", table.Left)
	for _, d := range rn.dates {
		row.AddNumber(sides[0].totals[amounts2.Key{Date: d}].Sub(sides[1].totals[amounts2.Key{Date: d}]))
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

type extractor struct {
	Amounts       amounts2.Amounts
	SortCommodity *journal.Commodity

	dates                   []time.Time
	accounts                []*journal.Account
	commodities, valuations []*journal.Commodity
}

func (ext *extractor) Init(jctx journal.Context) {
	dates := make(map[time.Time]struct{})
	leafs := make(map[*journal.Account]struct{})
	valuations := make(map[*journal.Commodity]struct{})
	commodities := make(map[*journal.Commodity]struct{})

	accountWeights := make(map[*journal.Account]float64)

	for k, v := range ext.Amounts {
		dates[k.Date] = struct{}{}
		leafs[k.Account] = struct{}{}
		valuations[k.Valuation] = struct{}{}
		commodities[k.Commodity] = struct{}{}
		if k.Valuation != nil && k.Valuation == ext.SortCommodity {
			f, _ := v.Abs().Float64()
			accountWeights[k.Account] += f
		} else {
			// default: sort alphabetically
			accountWeights[k.Account] = 0
		}
	}

	// sort dates
	for d := range dates {
		ext.dates = append(ext.dates, d)
	}
	sort.Slice(ext.dates, func(i, j int) bool { return ext.dates[i].Before(ext.dates[j]) })

	// fill parents and sort accounts
	accounts := make(map[*journal.Account]struct{})
	for a := range leafs {
		accounts[a] = struct{}{}
		for p := a.Parent(); p != nil; p = p.Parent() {
			accounts[p] = struct{}{}
			accountWeights[p] += accountWeights[a]
		}
	}
	for _, a := range jctx.Accounts().SortedPreOrder(accountWeights) {
		if _, ok := accounts[a]; ok {
			ext.accounts = append(ext.accounts, a)
		}
	}

	for a := range commodities {
		ext.commodities = append(ext.commodities, a)
	}
	sort.Slice(ext.commodities, func(i, j int) bool { return ext.commodities[i].String() < ext.commodities[j].String() })

	for a := range valuations {
		ext.valuations = append(ext.valuations, a)
	}
	sort.Slice(ext.valuations, func(i, j int) bool {
		// valuation can be nil
		v0, v1 := ext.valuations[i], ext.valuations[j]
		return v1 != nil && (v0 == nil || ext.valuations[i].String() < ext.valuations[j].String())
	})
}
