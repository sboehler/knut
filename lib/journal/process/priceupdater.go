package process

import (
	"context"

	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
)

// PriceUpdater updates the prices in a stream of days.
type PriceUpdater struct {
	Valuation *journal.Commodity
}

// Process computes prices.
func (pu PriceUpdater) Process(ctx context.Context, inCh <-chan *ast.Day, outCh chan<- *ast.Day) error {
	previous := new(ast.Day)
	prc := make(journal.Prices)
	return cpr.Consume(ctx, inCh, func(day *ast.Day) error {
		if pu.Valuation != nil {
			if len(day.Prices) == 0 {
				day.Normalized = previous.Normalized
			} else {
				for _, p := range day.Prices {
					prc.Insert(p.Commodity, p.Price, p.Target)
				}
				day.Normalized = prc.Normalize(pu.Valuation)
			}
		}
		previous = day
		return cpr.Push(ctx, outCh, day)
	})
}
