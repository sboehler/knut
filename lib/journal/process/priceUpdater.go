package process

import (
	"context"

	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/past"
	"github.com/sboehler/knut/lib/journal/prices"
	"github.com/sboehler/knut/lib/journal/val"
)

// PriceUpdater updates the prices in a stream of days.
type PriceUpdater struct {
	Context   journal.Context
	Valuation *journal.Commodity
}

// ProcessStream computes prices.
func (pr PriceUpdater) ProcessStream(ctx context.Context, inCh <-chan *past.Day) (<-chan *val.Day, <-chan error) {
	var (
		resCh = make(chan *val.Day, 100)
		errCh = make(chan error)
		prc   = make(prices.Prices)
	)
	go func() {
		defer close(resCh)
		defer close(errCh)

		var previous *val.Day
		for day := range inCh {
			var npr prices.NormalizedPrices
			if pr.Valuation != nil {
				if day.AST != nil && len(day.AST.Prices) > 0 {
					for _, p := range day.AST.Prices {
						prc.Insert(p)
					}
					npr = prc.Normalize(pr.Valuation)
				} else if previous == nil {
					npr = prc.Normalize(pr.Valuation)
				} else {
					npr = previous.Prices
				}
			}
			vday := &val.Day{
				Date:   day.Date,
				Day:    day,
				Prices: npr,
			}
			select {
			case resCh <- vday:
				previous = vday
			case <-ctx.Done():
				return
			}
		}
	}()
	return resCh, errCh
}
