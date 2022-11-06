package process

import (
	"context"

	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/journal"
)

// PriceUpdater updates the prices in a stream of days.
type PriceUpdater struct {
	Valuation *journal.Commodity
}

// Process computes prices.
func (pu PriceUpdater) Process(ctx context.Context, inCh <-chan *journal.Day, outCh chan<- *journal.Day) error {
	if pu.Valuation == nil {
		return cpr.Consume(ctx, inCh, func(d *journal.Day) error {
			return cpr.Push(ctx, outCh, d)
		})
	}
	var previous journal.NormalizedPrices
	prc := make(journal.Prices)
	return cpr.Consume(ctx, inCh, func(day *journal.Day) error {
		if len(day.Prices) == 0 {
			day.Normalized = previous
		} else {
			for _, p := range day.Prices {
				prc.Insert(p.Commodity, p.Price, p.Target)
			}
			day.Normalized = prc.Normalize(pu.Valuation)
			previous = day.Normalized
		}
		return cpr.Push(ctx, outCh, day)
	})
}
