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

	"github.com/shopspring/decimal"

	"github.com/sboehler/knut/lib/amount"
	"github.com/sboehler/knut/lib/balance"
	"github.com/sboehler/knut/lib/model/accounts"
	"github.com/sboehler/knut/lib/model/commodities"
)

// Report is a balance report for a range of dates.
type Report struct {
	Dates       []time.Time
	Segments    map[accounts.AccountType]*Segment
	Commodities []*commodities.Commodity
	Positions   map[*commodities.Commodity]amount.Vec
}

// Position is a position.
type Position struct {
	balance.CommodityAccount
	Amounts amount.Vec
}

// Builder contains configuration options to create a report.
type Builder struct {
	Value    bool
	Collapse []Collapse
}

// Collapse is a rule for collapsing (shortening) accounts.
type Collapse struct {
	Level int
	Regex *regexp.Regexp
}

// MatchAccount determines whether this Collapse matches the
// given Account.
func (c Collapse) MatchAccount(a *accounts.Account) bool {
	return c.Regex == nil || c.Regex.MatchString(a.String())
}

// Build creates a new report.
func (b Builder) Build(bal []*balance.Balance) (*Report, error) {
	// compute the dates and positions array
	var (
		dates     = make([]time.Time, 0, len(bal))
		positions = make([]map[balance.CommodityAccount]amount.Amount, 0, len(bal))
	)
	for _, b := range bal {
		dates = append(dates, b.Date)
		positions = append(positions, b.Positions)
	}
	var (
		// collect arrays of amounts by commodity account, across balances
		sortedPos = mergePositions(b.Value, positions)

		//compute the segments
		segments = buildSegments(b, sortedPos)

		// compute totals
		totals = make(map[*commodities.Commodity]amount.Vec)
	)
	for _, s := range segments {
		s.sum(totals)
	}

	// compute sorted commodities
	var commodities = make([]*commodities.Commodity, 0, len(totals))
	for c := range totals {
		commodities = append(commodities, c)
	}
	sort.Slice(commodities, func(i, j int) bool {
		return commodities[i].String() < commodities[j].String()
	})

	return &Report{
		Dates:       dates,
		Commodities: commodities,
		Segments:    segments,
		Positions:   totals,
	}, nil
}

func mergePositions(value bool, positions []map[balance.CommodityAccount]amount.Amount) []Position {
	var commodityAccounts = make(map[balance.CommodityAccount]bool)
	for _, p := range positions {
		for ca := range p {
			commodityAccounts[ca] = true
		}
	}
	var res = make([]Position, 0, len(commodityAccounts))
	for ca := range commodityAccounts {
		var (
			vec   = amount.NewVec(len(positions))
			empty = true
		)
		for i, p := range positions {
			if amount, exists := p[ca]; exists {
				var val decimal.Decimal
				if value {
					val = amount.Value()
				} else {
					val = amount.Amount()
				}
				if !val.IsZero() {
					vec.Values[i] = val
					empty = false
				}
			}
		}
		if empty {
			continue
		}
		res = append(res, Position{
			CommodityAccount: ca,
			Amounts:          vec,
		})
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].Less(res[j].CommodityAccount)
	})
	return res
}

func buildSegments(o Builder, positions []Position) map[accounts.AccountType]*Segment {
	var result = make(map[accounts.AccountType]*Segment)
	for _, position := range positions {
		var (
			at = position.Account.Type()
			k  = shorten(o.Collapse, position.Account)
		)
		// Any positions with zero keys should end up in totals.
		if len(k) > 0 {
			var s, ok = result[at]
			if !ok {
				s = NewSegment(at.String())
				result[at] = s
			}
			s.insert(k[1:], position)
		}
	}
	return result
}

// shorten shortens the given account according to the given rules.
func shorten(c []Collapse, a *accounts.Account) []string {
	var s = a.Split()
	for _, c := range c {
		if c.MatchAccount(a) && len(s) > c.Level {
			s = s[:c.Level]
		}
	}
	return s
}
