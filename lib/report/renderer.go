// Copyright 2020 Silvio Böhler
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
	"github.com/sboehler/knut/lib/amount"
	"github.com/sboehler/knut/lib/table"
)

// Renderer renders a report.
type Renderer struct {
	table           *table.Table
	indent          int
	showCommodities bool
	report          *Report
}

// NewRenderer creates a new report renderer.
func NewRenderer(showCommodities bool) *Renderer {
	return &Renderer{
		showCommodities: showCommodities,
	}
}

// Render renders a report.
func (rn *Renderer) Render(r *Report) *table.Table {

	rn.table = table.New(1, len(r.Dates))
	rn.indent = 0
	rn.report = r

	// sep
	rn.table.AddSeparatorRow()

	// header
	header := rn.table.AddRow().AddText("Account", table.Center)
	for _, d := range r.Dates {
		header.AddText(d.Format("2006-01-02"), table.Center)
	}

	// sep
	rn.table.AddSeparatorRow()

	// values
	for _, s := range r.Segments {
		if rn.showCommodities {
			rn.renderSegmentWithCommodities(s)
		} else {
			rn.renderSegment(s)
		}
		rn.table.AddSeparatorRow()
	}
	// totals
	if rn.showCommodities {
		rn.renderSegmentWithCommodities(&Segment{
			Key:       "Total",
			Positions: r.Positions,
		})
	} else {
		rn.renderSegment(&Segment{
			Key:       "Total",
			Positions: r.Positions,
		})
	}
	rn.table.AddSeparatorRow()

	return rn.table
}

func (rn *Renderer) renderSegment(s *Segment) {
	header := rn.table.AddRow().AddIndented(s.Key, rn.indent)

	// compute total value
	total := amount.NewVec(len(rn.report.Dates))
	for _, amounts := range s.Positions {
		total.Add(amounts)
	}
	// fill header cells with total values
	for _, amount := range total.Values {
		a := amount.Round(2)
		if a.IsZero() {
			header.AddEmpty()
		} else {
			header.AddText(a.String(), table.Right)
		}
	}

	// render subsegments
	rn.indent = rn.indent + 2
	for _, ss := range s.Subsegments {
		rn.renderSegment(ss)
	}
	rn.indent = rn.indent - 2
}

func (rn *Renderer) renderSegmentWithCommodities(segment *Segment) {
	header := rn.table.AddRow().AddIndented(segment.Key, rn.indent)
	for range rn.report.Dates {
		header.AddEmpty()
	}

	// add one row per commodity in this position
	rn.indent = rn.indent + 2
	for _, commodity := range rn.report.Commodities {
		if amounts, ok := segment.Positions[commodity]; ok {
			row := rn.table.AddRow().AddIndented(commodity.String(), rn.indent)
			for _, amount := range amounts.Values {
				v := amount.Round(2)
				if v.IsZero() {
					row.AddEmpty()
				} else {
					row.AddText(v.String(), table.Right)
				}
			}
		}
	}

	// render subsegments
	for _, ss := range segment.Subsegments {
		rn.renderSegmentWithCommodities(ss)
	}
	rn.indent = rn.indent - 2
}
