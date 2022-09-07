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
		rn.table.AddEmptyRow()
		rn.render(0, n)
	}
	for _, n := range rn.report.EIE.children {
		rn.table.AddEmptyRow()
		rn.render(0, n)
	}
	rn.table.AddSeparatorRow()

	return rn.table
}

func (rn *Renderer) render(indent int, n *Node) {
	vals := n.Amounts.SumBy(nil, amounts.KeyMapper{
		Date:      amounts.Identity[time.Time],
		Commodity: amounts.Identity[*journal.Commodity],
	}.Build())
	row := rn.table.AddRow().AddIndented(n.Account.Segment(), indent)
	for i, c := range vals.CommoditiesSorted() {
		if c != nil {
			if i == 0 {
				row.FillEmpty()
			}
			row = rn.table.AddRow().AddIndented(c.String(), indent+2)
		}
		for _, d := range rn.Dates {
			v := vals[amounts.DateCommodityKey(d, c)]
			if !n.Account.IsAL() {
				v = v.Neg()
			}
			if v.IsZero() {
				row.AddEmpty()
			} else {
				row.AddNumber(v)
			}
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
