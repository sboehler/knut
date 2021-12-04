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
	Value   bool
	Mapping ledger.Mapping
}

// Build creates a new report.
func (b Builder) Build(bs []*balance.Balance) (*Report, error) {
	// compute the dates and positions array
	var (
		dates     []time.Time
		positions []Position
	)
	for i, bal := range bs {
		dates = append(dates, bal.Date)
		var bp map[balance.CommodityAccount]decimal.Decimal
		if b.Value {
			bp = bal.Values
		} else {
			bp = bal.Amounts
		}
		for ca, pos := range bp {
			index := sort.Search(len(positions), func(i int) bool { return !positions[i].CommodityAccount.Less(ca) })
			if index == len(positions) || positions[index].CommodityAccount != ca {
				positions = append(positions, Position{})
				copy(positions[index+1:], positions[index:])
				positions[index] = Position{ca, vector.New(len(bs))}
			}
			positions[index].Amounts.Values[i] = pos
		}
	}
	var (
		//compute the segments
		segments = b.buildSegments(positions)

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

func (b Builder) buildSegments(positions []Position) map[ledger.AccountType]*Segment {
	var result = make(map[ledger.AccountType]*Segment)
	for _, position := range positions {
		var (
			at = position.Account.Type()
			// TODO: get rid of segments by using accounts hierarchy.
			k = position.Account.Map(b.Mapping).Split()
		)
		// ignore positions with zero keys
		if len(k) == 0 {
			continue
		}
		s, ok := result[at]
		if !ok {
			s = NewSegment(at.String())
			result[at] = s
		}
		s.insert(k[1:], position)
	}
	return result
}
