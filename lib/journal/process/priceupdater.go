package process

import (
	"context"

	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
)

// PriceUpdater updates the prices in a stream of days.
type PriceUpdater struct {
	Context   journal.Context
	Valuation *journal.Commodity
}

// Process computes prices.
func (pu PriceUpdater) Process(ctx context.Context, inCh <-chan *ast.Day, outCh chan<- *ast.Day) error {
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
		if err := cpr.Push(ctx, outCh, day); err != nil {
			return err
		}
	}
	return nil

}
