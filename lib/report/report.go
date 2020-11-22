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
	"regexp"
	"sort"
	"time"

	"github.com/sboehler/knut/lib/amount"
	"github.com/sboehler/knut/lib/balance"
	"github.com/sboehler/knut/lib/model"
	"github.com/sboehler/knut/lib/model/accounts"
	"github.com/sboehler/knut/lib/model/commodities"

	"github.com/shopspring/decimal"
)

// Report is a balance report for a range of dates.
type Report struct {
	Dates       []time.Time
	Options     Options
	Segments    []*Segment
	Commodities []*commodities.Commodity
	Positions   map[*commodities.Commodity]amount.Vec
}

// Options contains configuration options to create a report.
type Options struct {
	Valuation *int
	Collapse  []Collapse
}

// Collapse is a rule for collapsing (shortening) accounts.
type Collapse struct {
	Level int
	Regex string
}

// NewReport creates a new report.
func NewReport(options Options, bal []*balance.Balance) (*Report, error) {
	// compute the dates and positions array
	dates := make([]time.Time, 0, len(bal))
	positions := make([]map[model.CommodityAccount]decimal.Decimal, 0, len(bal))
	for _, b := range bal {
		dates = append(dates, b.Date)
		positions = append(positions, b.GetPositions(options.Valuation))
	}
	// collect arrays of amounts by commodity account, across balances
	sortedPos := mergePositions(positions)
	// compute the segments
	segment, err := buildSegments(options, sortedPos)
	if err != nil {
		return nil, err
	}
	// compute totals
	totals := segment.sum(nil)

	// compute sorted commodities
	commodities := make([]*commodities.Commodity, 0, len(totals))
	for c := range totals {
		commodities = append(commodities, c)
	}
	sort.Slice(commodities, func(i, j int) bool {
		return commodities[i].String() < commodities[j].String()
	})

	return &Report{
		Dates:       dates,
		Commodities: commodities,
		Options:     options,
		Segments:    segment.Subsegments,
		Positions:   totals,
	}, nil
}

func mergePositions(positions []map[model.CommodityAccount]decimal.Decimal) []model.Position {
	commodityAccounts := make(map[model.CommodityAccount]bool)
	for _, p := range positions {
		for ca := range p {
			commodityAccounts[ca] = true
		}
	}
	res := make([]model.Position, 0, len(commodityAccounts))
	for ca := range commodityAccounts {
		vec := amount.NewVec(len(positions))
		for i, p := range positions {
			if value, exists := p[ca]; exists {
				vec.Values[i] = value
			}
		}
		res = append(res, model.Position{
			CommodityAccount: ca,
			Amounts:          vec,
		})
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].Less(res[j].CommodityAccount)
	})
	return res
}

func buildSegments(o Options, positions []model.Position) (*Segment, error) {
	toplevel := NewSegment("")
	for _, position := range positions {
		k, err := shorten(o.Collapse, position.Account())
		if err != nil {
			return nil, err
		}
		// Any positions with zero keys should end up in totals.
		if len(k) > 0 {
			toplevel.insert(k, position)
		}
	}
	return toplevel, nil
}

// shorten shortens the given account according to the given rules.
func shorten(c []Collapse, a *accounts.Account) ([]string, error) {
	s := a.Split()
	for _, c := range c {
		matched, err := regexp.MatchString(c.Regex, a.String())
		if err != nil {
			return nil, err
		}
		if matched && len(s) > c.Level {
			s = s[:c.Level]
		}
	}
	return s, nil
}
