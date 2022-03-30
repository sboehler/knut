package process

import (
	"context"

	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
	"golang.org/x/sync/errgroup"
)

// PriceUpdater updates the prices in a stream of days.
type PriceUpdater struct {
	Context   journal.Context
	Valuation *journal.Commodity
}

// Process2 computes prices.
func (pu PriceUpdater) Process2(ctx context.Context, g *errgroup.Group, inCh <-chan *ast.Day) <-chan *ast.Day {
	resCh := make(chan *ast.Day, 100)
	g.Go(func() error {
		defer close(resCh)
		var (
			prc      = make(journal.Prices)
			previous *ast.Day
		)
		for {
			day, ok, err := cpr.Pop(ctx, inCh)
			if err != nil {
				return err
			}
			if !ok {
				break
			}
			if pu.Valuation != nil {
				if len(day.Prices) > 0 {
					for _, p := range day.Prices {
						prc.Insert(p.Commodity, p.Price, p.Target)
					}
					day.Normalized = prc.Normalize(pu.Valuation)
				} else if previous == nil {
					day.Normalized = prc.Normalize(pu.Valuation)
				} else {
					day.Normalized = previous.Normalized
				}
			}
			previous = day
			if err := cpr.Push(ctx, resCh, day); err != nil {
				return err
			}
		}
		return nil
	})
	return resCh
}
