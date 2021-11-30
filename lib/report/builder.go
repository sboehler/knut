package report

import (
	"sort"
	"time"

	"github.com/sboehler/knut/lib/balance"
	"github.com/sboehler/knut/lib/ledger"
	"github.com/sboehler/knut/lib/vector"
	"github.com/shopspring/decimal"
)

// Builder contains configuration options to build a report.
type Builder struct {
	Value    bool
	Collapse []Collapse
}

// Build creates a new report.
func (b Builder) Build(bal []*balance.Balance) (*Report, error) {
	// compute the dates and positions array
	var (
		dates     = make([]time.Time, 0, len(bal))
		positions = make([]map[balance.CommodityAccount]decimal.Decimal, 0, len(bal))
	)
	for _, ba := range bal {
		dates = append(dates, ba.Date)
		if b.Value {
			positions = append(positions, ba.Values)
		} else {
			positions = append(positions, ba.Amounts)
		}
	}
	var (
		// collect arrays of amounts by commodity account, across balances
		sortedPos = b.mergePositions(positions)

		//compute the segments
		segments = b.buildSegments(sortedPos)

		// compute totals
		totals = make(map[*ledger.Commodity]vector.Vector)
	)
	for _, s := range segments {
		s.sum(totals)
	}

	// compute sorted commodities
	var commodities = make([]*ledger.Commodity, 0, len(totals))
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

func (Builder) mergePositions(positions []map[balance.CommodityAccount]decimal.Decimal) []Position {
	var commodityAccounts = make(map[balance.CommodityAccount]bool)
	for _, p := range positions {
		for ca := range p {
			commodityAccounts[ca] = true
		}
	}
	var res = make([]Position, 0, len(commodityAccounts))
	for ca := range commodityAccounts {
		var (
			vec   = vector.New(len(positions))
			empty = true
		)
		for i, p := range positions {
			if amount, exists := p[ca]; exists {
				if !amount.IsZero() {
					vec.Values[i] = amount
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

func (b Builder) buildSegments(positions []Position) map[ledger.AccountType]*Segment {
	var result = make(map[ledger.AccountType]*Segment)
	for _, position := range positions {
		var (
			at = position.Account.Type()
			k  = b.shorten(b.Collapse, position.Account)
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
func (Builder) shorten(c []Collapse, a *ledger.Account) []string {
	var s = a.Split()
	for _, c := range c {
		if c.MatchAccount(a) && len(s) > c.Level {
			s = s[:c.Level]
		}
	}
	return s
}
