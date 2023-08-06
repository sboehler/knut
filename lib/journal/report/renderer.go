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

	"github.com/sboehler/knut/lib/common/date"
	"github.com/sboehler/knut/lib/common/mapper"
	"github.com/sboehler/knut/lib/common/regex"
	"github.com/sboehler/knut/lib/common/table"
	"github.com/sboehler/knut/lib/journal"
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

	commoditiesColumn bool
	partition         date.Partition
}

// Render renders a report.
func (rn *Renderer) Render(r *Report) *table.Table {
	rn.commoditiesColumn = rn.Valuation == nil || len(rn.CommodityDetails) > 0
	rn.partition = r.dates
	if !rn.SortAlphabetically {
		r.ComputeWeights()
	}
	var tbl *table.Table
	if rn.commoditiesColumn {
		tbl = table.New(1, 1, rn.partition.Size())
	} else {
		tbl = table.New(1, rn.partition.Size())
	}
	tbl.AddSeparatorRow()
	header := tbl.AddRow().AddText("Account", table.Center)
	if rn.commoditiesColumn {
		header.AddText("Comm", table.Center)
	}
	for _, d := range rn.partition.EndDates() {
		header.AddText(d.Format("2006-01-02"), table.Center)
	}
	tbl.AddSeparatorRow()

	totalAL, totalEIE := r.Totals(journal.KeyMapper{
		Date:      mapper.Identity[time.Time],
		Commodity: commodity.Map(rn.Valuation == nil),
	}.Build())

	for _, n := range r.AL.Children() {
		rn.renderNode(tbl, 0, n)
		tbl.AddEmptyRow()
	}
	rn.render(tbl, 0, "Total (A+L)", false, totalAL)
	tbl.AddSeparatorRow()
	for _, n := range r.EIE.Children() {
		rn.renderNode(tbl, 0, n)
		tbl.AddEmptyRow()
	}
	rn.render(tbl, 0, "Total (E+I+E)", true, totalEIE)
	tbl.AddSeparatorRow()
	totalAL.Plus(totalEIE)
	rn.render(tbl, 0, "Delta", false, totalAL)
	tbl.AddSeparatorRow()

	return tbl
}

func (rn *Renderer) renderNode(t *table.Table, indent int, n *Node) {
	if n.Account != nil {
		showCommodities := rn.Valuation == nil || rn.CommodityDetails.MatchString(n.Account.Name())
		vals := n.Amounts.SumBy(nil, journal.KeyMapper{
			Date:      mapper.Identity[time.Time],
			Commodity: commodity.Map(showCommodities),
		}.Build())
		rn.render(t, indent, n.Account.Segment(), !n.Account.IsAL(), vals)
	}
	for _, ch := range n.Children() {
		rn.renderNode(t, indent+2, ch)
	}
}

func (rn *Renderer) render(t *table.Table, indent int, name string, neg bool, vals journal.Amounts) {
	if len(vals) == 0 {
		t.AddRow().AddIndented(name, indent).FillEmpty()
		return
	}
	for i, c := range vals.CommoditiesSorted() {
		row := t.AddRow()
		if i == 0 {
			row.AddIndented(name, indent)
		} else {
			row.AddEmpty()
		}
		if rn.commoditiesColumn {

			if c != nil {
				row.AddText(c.Name(), table.Left)
			} else if rn.Valuation != nil {
				row.AddText(rn.Valuation.Name(), table.Left)
			} else {
				row.AddEmpty()
			}
		}
		var total decimal.Decimal
		for _, d := range rn.partition.EndDates() {
			v := vals[journal.DateCommodityKey(d, c)]
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
