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
	"github.com/sboehler/knut/lib/ledger"
	"github.com/sboehler/knut/lib/table"
)

// Renderer renders a report.
type Renderer struct {
	Commodities bool
	negate      bool
	report      *Report
	table       *table.Table
}

// Render renders a report.
func (rn *Renderer) Render(r *Report) *table.Table {

	rn.table = table.New(1, len(r.Dates))
	rn.report = r

	var render = rn.render
	if rn.Commodities {
		render = rn.renderByCommodity
	}

	rn.table.AddSeparatorRow()

	var header = rn.table.AddRow().AddText("Account", table.Center)
	for _, d := range r.Dates {
		header.AddText(d.Format("2006-01-02"), table.Center)
	}
	rn.table.AddSeparatorRow()

	var (
		subtree = rn.report.Subtree()
		al, eie []*ledger.Account
	)
	for acc := range rn.report.Context.Accounts().PreOrder() {
		if _, ok := subtree[acc]; !ok {
			continue
		}
		if acc.Type() == ledger.ASSETS || acc.Type() == ledger.LIABILITIES {
			al = append(al, acc)
		} else {
			eie = append(eie, acc)
		}
	}

	alTotals := make(indexByCommodity)
	if len(al) > 0 {
		for i, acc := range al {
			if i != 0 && acc.Level() == 1 {
				rn.table.AddEmptyRow()
			}
			render(2*(acc.Level()-1), acc.Segment(), rn.report.Positions[acc])
			alTotals.AddOther(rn.report.Positions[acc])
		}
		rn.table.AddEmptyRow()
		render(0, "Total", alTotals)
		rn.table.AddSeparatorRow()
	}
	eieTotals := make(indexByCommodity)
	if len(eie) > 0 {
		rn.negate = true
		for i, acc := range eie {
			if i != 0 && acc.Level() == 1 {
				rn.table.AddEmptyRow()
			}
			render(2*(acc.Level()-1), acc.Segment(), rn.report.Positions[acc])
			eieTotals.AddOther(rn.report.Positions[acc])
		}
		rn.table.AddEmptyRow()
		render(0, "Total", eieTotals)
		rn.table.AddSeparatorRow()
		rn.negate = false
	}

	alTotals.AddOther(eieTotals)
	render(0, "Delta", alTotals)
	rn.table.AddSeparatorRow()
	return rn.table
}

func (rn *Renderer) render(indent int, key string, pos indexByCommodity) {
	total := make(indexByDate)
	for _, amounts := range pos {
		for d, val := range amounts {
			total[d] = total[d].Add(val)
		}
	}

	// fill header cells with total values
	var header = rn.table.AddRow().AddIndented(key, indent)
	for _, date := range rn.report.Dates {
		amount, ok := total[date]
		if !ok || amount.IsZero() {
			header.AddEmpty()
		} else {
			if rn.negate {
				amount = amount.Neg()
			}
			header.AddNumber(amount)
		}
	}
}

func (rn *Renderer) renderByCommodity(indent int, key string, pos indexByCommodity) {
	rn.table.AddRow().AddIndented(key, indent).FillEmpty()
	for commodity := range rn.report.Context.Commodities().Enumerate() {
		amounts, ok := pos[commodity]
		if !ok {
			continue
		}
		var row = rn.table.AddRow().AddIndented(commodity.String(), indent+1)
		for _, date := range rn.report.Dates {
			amount, ok := amounts[date]
			if !ok || amount.IsZero() {
				row.AddEmpty()
			} else {
				if rn.negate {
					amount = amount.Neg()
				}
				row.AddNumber(amount)
			}
		}
	}
}
