package process

import (
	"context"

	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
)

// PeriodDiffer filters the incoming days according to the dates
// specified.
type PeriodDiffer struct {
	Valuation *journal.Commodity
}

// Process does the diffing.
func (pf PeriodDiffer) Process(ctx context.Context, inCh <-chan *ast.Period, outCh chan<- *ast.Period) error {
	for {
		d, ok, err := cpr.Pop(ctx, inCh)
		if err != nil {
			return err
		}
		if !ok {
			break
		}
		if d.Amounts != nil || d.PrevAmounts != nil {
			d.DeltaAmounts = d.Amounts.Clone().Minus(d.PrevAmounts)
		}
		if d.Values != nil || d.PrevValues != nil {
			d.DeltaValues = d.Values.Clone().Minus(d.PrevValues)
		}
		if err := cpr.Push(ctx, outCh, d); err != nil {
			return err
		}
	}
	return nil
}
