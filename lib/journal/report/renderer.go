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
	"math"
	"sort"
	"time"

	"github.com/sboehler/knut/lib/common/table"
	"github.com/sboehler/knut/lib/journal"
)

// Renderer renders a report.
type Renderer struct {
	Context            journal.Context
	ShowCommodities    bool
	SortAlphabetically bool

	report *Balance
	table  *table.Table
	dates  []time.Time
}

// Render renders a report.
func (rn *Renderer) Render(r *Balance) *table.Table {
	rn.report = r
	for d := range r.Dates {
		rn.dates = append(rn.dates, d)
	}
	sort.Slice(rn.dates, func(i, j int) bool {
		return rn.dates[i].Before(rn.dates[j])
	})
	rn.table = table.New(1, len(rn.dates))
	rn.table.AddSeparatorRow()

	header := rn.table.AddRow().AddText("Account", table.Center)
	for _, d := range rn.dates {
		header.AddText(d.Format("2006-01-02"), table.Center)
	}
	rn.table.AddSeparatorRow()

	weights := rn.computeWeights()

	var (
		subtree = rn.report.Subtree()
		al, eie []*journal.Account
	)
	for _, acc := range rn.Context.Accounts().SortedPreOrder(weights) {
		if _, ok := subtree[acc]; !ok {
			continue
		}
		if acc.IsAL() {
			al = append(al, acc)
		} else {
			eie = append(eie, acc)
		}
	}

	alTotals := rn.renderSection(al, false)
	eieTotals := rn.renderSection(eie, true)
	alTotals.AddFrom(eieTotals)
	alTotals.Normalize()
	rn.render(0, "Delta", false, alTotals)
	rn.table.AddSeparatorRow()
	return rn.table
}

func (rn *Renderer) computeWeights() map[*journal.Account]float64 {
	weights := make(map[*journal.Account]float64)
	for _, acc := range rn.Context.Accounts().PostOrder() {
		var w float64
		if !rn.SortAlphabetically {

			for _, ibd := range rn.report.Positions[acc] {
				for _, v := range ibd {
					f, _ := v.Float64()
					w += f
				}
			}
			for _, chAcc := range acc.Children() {
				w += weights[chAcc]
			}
		}
		weights[acc] = math.Abs(w)
	}
	return weights
}

func (rn *Renderer) renderSection(al []*journal.Account, negate bool) indexByCommodity {
	res := make(indexByCommodity)
	if len(al) == 0 {
		return res
	}
	for i, acc := range al {
		if i != 0 && acc.Level() == 1 {
			rn.table.AddEmptyRow()
		}
		rn.render(2*(acc.Level()-1), acc.Segment(), !acc.IsAL(), rn.report.Positions[acc])
		res.AddFrom(rn.report.Positions[acc])
	}
	res.Normalize()
	rn.table.AddEmptyRow()
	rn.render(0, "Total", negate, res)
	rn.table.AddSeparatorRow()
	return res
}

func (rn *Renderer) render(indent int, key string, negate bool, byCommodity indexByCommodity) {
	if rn.ShowCommodities {
		rn.renderByCommodity(indent, key, negate, byCommodity)
	} else {
		rn.renderAmounts(indent, key, negate, byCommodity.Sum())
	}
}

func (rn *Renderer) renderByCommodity(indent int, key string, negate bool, byCommodity indexByCommodity) {
	rn.table.AddRow().AddIndented(key, indent).FillEmpty()
	for commodity := range rn.Context.Commodities().Enumerate() {
		if byDate, ok := byCommodity[commodity]; ok {
			rn.renderAmounts(indent+2, commodity.String(), negate, byDate)
		}
	}
}

func (rn Renderer) renderAmounts(indent int, key string, negate bool, byDate indexByDate) {
	row := rn.table.AddRow().AddIndented(key, indent)
	for _, date := range rn.dates {
		amount, ok := byDate[date]
		if !ok || amount.IsZero() {
			row.AddEmpty()
			continue
		}
		if negate {
			amount = amount.Neg()
		}
		row.AddNumber(amount)
	}
}
