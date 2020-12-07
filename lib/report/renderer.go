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
	"github.com/sboehler/knut/lib/model/accounts"
	"github.com/sboehler/knut/lib/model/commodities"
	"github.com/sboehler/knut/lib/table"
)

// Renderer renders a report.
type Renderer struct {
	// the configuration of this Renderer
	config Config
	// the report which is to be rendered
	negate bool
	report *Report
	// the table which is being built
	table *table.Table
	// the current indentation level
	indent int
}

// Config configures a Renderer.
type Config struct {
	Commodities bool
}

const indent = 2

// NewRenderer creates a new report renderer.
func NewRenderer(config Config) *Renderer {
	return &Renderer{config: config}
}

// Render renders a report.
func (rn *Renderer) Render(r *Report) *table.Table {

	rn.table = table.New(1, len(r.Dates))
	rn.indent = 0
	rn.report = r

	var render func(s *Segment)
	if rn.config.Commodities {
		render = rn.renderSegmentWithCommodities
	} else {
		render = rn.renderSegment
	}

	// sep
	rn.table.AddSeparatorRow()

	// header
	header := rn.table.AddRow().AddText("Account", table.Center)
	for _, d := range r.Dates {
		header.AddText(d.Format("2006-01-02"), table.Center)
	}
	// sep
	rn.table.AddSeparatorRow()

	var g1, g2 []*Segment

	for _, at := range accounts.AccountTypes {
		s, ok := rn.report.Segments[at]
		if !ok {
			continue
		}
		if at == accounts.ASSETS || at == accounts.LIABILITIES {
			g1 = append(g1, s)
		} else {
			g2 = append(g2, s)
		}
	}

	// values
	if len(g1) > 0 {
		for _, s := range g1 {
			render(s)
			rn.table.AddEmptyRow()
		}

		totals := map[*commodities.Commodity]amount.Vec{}
		for _, s := range g1 {
			s.sum(totals)
		}
		render(&Segment{Key: "Total", Positions: totals})
		rn.table.AddSeparatorRow()

	}
	if len(g2) > 0 {
		rn.negate = true
		for _, s := range g2 {
			render(s)
			rn.table.AddEmptyRow()
		}
		totals := map[*commodities.Commodity]amount.Vec{}
		for _, s := range g2 {
			s.sum(totals)
		}
		render(&Segment{Key: "Total", Positions: totals})

		rn.negate = false
		rn.table.AddSeparatorRow()
	}

	// totals
	render(&Segment{
		Key:       "Delta",
		Positions: r.Positions,
	})
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
		if amount.IsZero() {
			header.AddEmpty()
		} else {
			if rn.negate {
				amount = amount.Neg()
			}
			header.AddNumber(amount)
		}
	}

	// render subsegments
	rn.indent += indent
	for _, ss := range s.Subsegments {
		rn.renderSegment(ss)
	}
	rn.indent -= indent
}

func (rn *Renderer) renderSegmentWithCommodities(segment *Segment) {
	header := rn.table.AddRow().AddIndented(segment.Key, rn.indent)
	for range rn.report.Dates {
		header.AddEmpty()
	}

	// add one row per commodity in this position
	rn.indent += indent
	for _, commodity := range rn.report.Commodities {
		if amounts, ok := segment.Positions[commodity]; ok {
			row := rn.table.AddRow().AddIndented(commodity.String(), rn.indent)
			for _, amount := range amounts.Values {
				if amount.IsZero() {
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

	// render subsegments
	for _, ss := range segment.Subsegments {
		rn.renderSegmentWithCommodities(ss)
	}
	rn.indent -= indent
}
