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

	"github.com/sboehler/knut/lib/common/amounts"
	"github.com/sboehler/knut/lib/common/mapper"
	"github.com/sboehler/knut/lib/common/table"
	"github.com/sboehler/knut/lib/journal"
	"github.com/shopspring/decimal"
)

// Renderer renders a report.
type Renderer struct {
	ShowCommodities    bool
	SortAlphabetically bool
	Dates              []time.Time
	Diff               bool

	table *table.Table
}

// Render renders a report.
func (rn *Renderer) Render(r *Report) *table.Table {
	if !rn.SortAlphabetically {
		r.ComputeWeights()
	}
	if rn.ShowCommodities {
		rn.table = table.New(1, 1, len(rn.Dates))
	} else {
		rn.table = table.New(1, len(rn.Dates))
	}
	rn.table.AddSeparatorRow()
	header := rn.table.AddRow().AddText("Account", table.Center)
	if rn.ShowCommodities {
		header.AddText("Comm", table.Center)
	}
	for _, d := range rn.Dates {
		header.AddText(d.Format("2006-01-02"), table.Center)
	}
	rn.table.AddSeparatorRow()

	totalAL, totalEIE := r.Totals()

	for _, n := range r.AL.Children() {
		rn.renderNode(0, n)
		rn.table.AddEmptyRow()
	}
	rn.render(0, "Total (A+L)", false, totalAL)
	rn.table.AddSeparatorRow()
	for _, n := range r.EIE.Children() {
		rn.renderNode(0, n)
		rn.table.AddEmptyRow()
	}
	rn.render(0, "Total (E+I+E)", true, totalEIE)
	rn.table.AddSeparatorRow()
	totalAL.Plus(totalEIE)
	rn.render(0, "Delta", false, totalAL)
	rn.table.AddSeparatorRow()

	return rn.table
}

func (rn *Renderer) renderNode(indent int, n *Node) {
	if n.Account != nil {
		vals := n.Amounts.SumBy(nil, amounts.KeyMapper{
			Date:      mapper.Identity[time.Time],
			Commodity: mapper.Identity[*journal.Commodity],
		}.Build())
		rn.render(indent, n.Account.Segment(), !n.Account.IsAL(), vals)
	}
	for _, ch := range n.Children() {
		rn.renderNode(indent+2, ch)
	}
}

func (rn *Renderer) render(indent int, name string, neg bool, vals amounts.Amounts) {
	if len(vals) == 0 {
		rn.table.AddRow().AddIndented(name, indent).FillEmpty()
		return
	}
	for i, c := range vals.CommoditiesSorted() {
		row := rn.table.AddRow()
		if i == 0 {
			row.AddIndented(name, indent)
		} else {
			row.AddEmpty()
		}
		if rn.ShowCommodities {
			row.AddText(c.Name(), table.Left)
		}
		var total decimal.Decimal
		for _, d := range rn.Dates {
			v := vals[amounts.DateCommodityKey(d, c)]
			if !rn.Diff {
				total = total.Add(v)
				v = total
			}
			if neg {
				v = v.Neg()
			}
			if v.IsZero() {
				row.AddEmpty()
			} else {
				row.AddNumber(v)
			}
		}
	}
}
