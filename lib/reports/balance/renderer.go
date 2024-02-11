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

package balance

import (
	"time"

	"github.com/sboehler/knut/lib/amounts"
	"github.com/sboehler/knut/lib/common/date"
	"github.com/sboehler/knut/lib/common/mapper"
	"github.com/sboehler/knut/lib/common/regex"
	"github.com/sboehler/knut/lib/common/table"
	"github.com/sboehler/knut/lib/model"
	"github.com/sboehler/knut/lib/model/commodity"
	"github.com/shopspring/decimal"
)

// Renderer renders a report.
type Renderer struct {
	Valuation          *model.Commodity
	CommodityDetails   regex.Regexes
	SortAlphabetically bool
	Diff               bool

	drawCommsColumn bool
	partition       date.Partition
}

// Render renders a report.
func (rn *Renderer) Render(r *Report) *table.Table {
	rn.drawCommsColumn = rn.Valuation == nil || len(rn.CommodityDetails) > 0
	rn.partition = r.partition
	r.SetAccounts()
	if rn.SortAlphabetically {
		r.SortAlpha()
	} else {
		r.SortWeighted()
	}
	var tbl *table.Table
	if rn.drawCommsColumn {
		tbl = table.New(1, 1, rn.partition.Size())
	} else {
		tbl = table.New(1, rn.partition.Size())
	}
	tbl.AddSeparatorRow()
	header := tbl.AddRow().AddText("Account", table.Center)
	if rn.drawCommsColumn {
		header.AddText("Comm", table.Center)
	}
	for _, d := range rn.partition.EndDates() {
		header.AddText(d.Format("2006-01-02"), table.Center)
	}
	tbl.AddSeparatorRow()

	totalAL, totalEIE := r.Totals(amounts.KeyMapper{
		Date:      mapper.Identity[time.Time],
		Commodity: commodity.IdentityIf(rn.Valuation == nil),
	}.Build())

	for _, n := range r.AL.Sorted {
		rn.renderNode(tbl, 0, false, n)
		tbl.AddEmptyRow()
	}
	rn.render(tbl, 0, "Total (A+L)", false, totalAL)
	tbl.AddSeparatorRow()
	for _, n := range r.EIE.Sorted {
		rn.renderNode(tbl, 0, true, n)
		tbl.AddEmptyRow()
	}
	rn.render(tbl, 0, "Total (E+I+E)", true, totalEIE)
	tbl.AddSeparatorRow()
	totalAL.Plus(totalEIE)
	rn.render(tbl, 0, "Delta", false, totalAL)
	tbl.AddSeparatorRow()

	return tbl
}

func (rn *Renderer) renderNode(t *table.Table, indent int, neg bool, n *Node) {
	var vals amounts.Amounts
	if n.Value.Account != nil {
		showCommodities := rn.Valuation == nil || rn.CommodityDetails.MatchString(n.Value.Account.Name())
		vals = n.Value.Amounts.SumBy(nil, amounts.KeyMapper{
			Date:      mapper.Identity[time.Time],
			Commodity: commodity.IdentityIf(showCommodities),
		}.Build())
	}
	if n.Segment != "" {
		rn.render(t, indent, n.Segment, neg, vals)
	}
	for _, ch := range n.Sorted {
		rn.renderNode(t, indent+2, neg, ch)
	}
}

func (rn *Renderer) render(t *table.Table, indent int, name string, neg bool, vals amounts.Amounts) {
	if len(vals) == 0 {
		t.AddRow().AddIndented(name, indent).FillEmpty()
		return
	}
	for i, commodity := range vals.CommoditiesSorted() {
		row := t.AddRow()
		if i == 0 {
			row.AddIndented(name, indent)
		} else {
			row.AddEmpty()
		}
		if rn.drawCommsColumn {
			if commodity != nil {
				row.AddText(commodity.Name(), table.Left)
			} else if rn.Valuation != nil {
				row.AddText(rn.Valuation.Name(), table.Left)
			} else {
				row.AddEmpty()
			}
		}
		var total decimal.Decimal
		for _, date := range rn.partition.EndDates() {
			v := vals[amounts.DateCommodityKey(date, commodity)]
			if !rn.Diff {
				total = total.Add(v)
				v = total
			}
			if neg {
				v = v.Neg()
			}
			row.AddDecimal(v)
		}
	}
}
