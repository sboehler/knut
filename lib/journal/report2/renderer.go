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

package report2

import (
	"time"

	"github.com/sboehler/knut/lib/common/amounts"
	"github.com/sboehler/knut/lib/common/table"
	"github.com/sboehler/knut/lib/journal"
)

// Renderer renders a report.
type Renderer struct {
	Context            journal.Context
	ShowCommodities    bool
	SortAlphabetically bool
	Dates              []time.Time

	report *Report
	table  *table.Table
}

// Render renders a report.
func (rn *Renderer) Render(r *Report) *table.Table {
	rn.report = r
	r.ComputeWeights()

	rn.table = table.New(1, len(rn.Dates))
	rn.table.AddSeparatorRow()
	header := rn.table.AddRow().AddText("Account", table.Center)
	for _, d := range rn.Dates {
		header.AddText(d.Format("2006-01-02"), table.Center)
	}
	rn.table.AddSeparatorRow()

	for _, n := range rn.report.AL.children {
		rn.render(0, n)
	}
	for _, n := range rn.report.EIE.children {
		rn.render(0, n)
	}
	// 	accs, weights := rn.accountWeights(rn.report)

	// 	idx := rn.report.Index(compare.Combine(
	// 		amounts.SortByAccount(rn.Context, weights),
	// 		amounts.SortByCommodity,
	// 		amounts.SortByDate,
	// 	))

	// 	header := rn.table.AddRow().AddText("Account", table.Center)
	// 	for _, d := range rn.dates {
	// 		header.AddText(d.End.Format("2006-01-02"), table.Center)
	// 	}
	// 	rn.table.AddSeparatorRow()

	// 	var (
	// 		subtree = rn.report.Subtree()
	// 		al, eie []*journal.Account
	// 	)
	// 	for _, acc := range rn.Context.Accounts().SortedPreOrder(weights) {
	// 		if _, ok := subtree[acc]; !ok {
	// 			continue
	// 		}
	// 		if acc.IsAL() {
	// 			al = append(al, acc)
	// 		} else {
	// 			eie = append(eie, acc)
	// 		}
	// 	}

	// 	alTotals := rn.renderSection(al, false)
	// 	eieTotals := rn.renderSection(eie, true)
	// 	alTotals.AddFrom(eieTotals)
	// 	alTotals.Normalize()
	// 	rn.render(0, "Delta", false, alTotals)
	// 	rn.table.AddSeparatorRow()
	return rn.table
}

// func (rn *Renderer) accountWeights(as amounts.Amounts) ([]*journal.Account, map[*journal.Account]float64) {
// 	weights := make(map[*journal.Account]float64)
// 	for k, v := range as {
// 		var f float64
// 		if k.Valuation != nil && !rn.SortAlphabetically {
// 			f, _ = v.Float64()
// 		}
// 		weights[k.Account] = f
// 	}
// 	res := make(map[*journal.Account]float64)
// 	for acc, w := range weights {
// 		for p := acc; p != nil; p = rn.Context.Accounts().Parent(acc) {
// 			res[p] += w
// 		}
// 	}
// 	var accs []*journal.Account
// 	for acc := range res {
// 		accs = append(accs, acc)
// 	}
// 	cmp := journal.CompareWeighted(rn.Context, res)
// 	sort.Slice(accs, func(i, j int) bool {
// 		return cmp(accs[i], accs[j]) == compare.Smaller
// 	})
// 	return accs, res
// }

// func (rn *Renderer) renderSection(al []*journal.Account, negate bool) indexByCommodity {
// 	res := make(indexByCommodity)
// 	if len(al) == 0 {
// 		return res
// 	}
// 	for i, acc := range al {
// 		if i != 0 && acc.Level() == 0 {
// 			rn.table.AddEmptyRow()
// 		}
// 		rn.render(2*(acc.Level()), acc.Segment(), !acc.IsAL(), rn.report.Positions[acc])
// 		res.AddFrom(rn.report.Positions[acc])
// 	}
// 	res.Normalize()
// 	rn.table.AddEmptyRow()
// 	rn.render(0, "Total", negate, res)
// 	rn.table.AddSeparatorRow()
// 	return res
// }

func (rn *Renderer) render(indent int, n *Node) {
	vals := n.Amounts.SumBy(nil, amounts.KeyMapper{
		Date:      amounts.Identity[time.Time],
		Commodity: amounts.Identity[*journal.Commodity],
	}.Build())
	row := rn.table.AddRow().AddIndented(n.Account.Segment(), indent)
	for _, d := range rn.Dates {
		v := vals[amounts.DateKey(d)]
		if !n.Account.IsAL() {
			v = v.Neg()
		}
		if v.IsZero() {
			row.AddEmpty()
		} else {
			row.AddNumber(v)
		}
	}
	for _, ch := range n.Children() {
		rn.render(indent+2, ch)
	}
}

// func (rn *Renderer) renderByCommodity(indent int, key string, negate bool, byCommodity indexByCommodity) {
// 	rn.table.AddRow().AddIndented(key, indent).FillEmpty()
// 	for _, commodity := range rn.Context.Commodities().All() {
// 		if byDate, ok := byCommodity[commodity]; ok {
// 			rn.renderAmounts(indent+2, commodity.String(), negate, byDate)
// 		}
// 	}
// }

// func (rn Renderer) renderAmounts(indent int, key string, negate bool, byDate indexByDate) {
// 	row := rn.table.AddRow().AddIndented(key, indent)
// 	for _, date := range rn.dates {
// 		amount, ok := byDate[date.End]
// 		if !ok || amount.IsZero() {
// 			row.AddEmpty()
// 			continue
// 		}
// 		if negate {
// 			amount = amount.Neg()
// 		}
// 		row.AddNumber(amount)
// 	}
// }
