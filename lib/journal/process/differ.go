package process

import (
	"context"

	"github.com/sboehler/knut/lib/common/amounts"
	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/journal/ast"
	"golang.org/x/sync/errgroup"
)

// Differ filters the incoming days according to the dates
// specified.
type Differ struct {
	Diff bool
}

// Process does the diffing.
func (pf Differ) Process(ctx context.Context, g *errgroup.Group, inCh <-chan *ast.Day) <-chan *ast.Day {
	if !pf.Diff {
		return inCh
	}
	resCh := make(chan *ast.Day, 100)

	g.Go(func() error {
		defer close(resCh)

		var prev amounts.Amounts
		for {
			d, ok, err := cpr.Pop(ctx, inCh)
			if err != nil {
				return err
			}
			if !ok {
				break
			}
			diff := d.Value.Clone().Minus(prev)

			prev = d.Value
			d.Value = diff
			if err := cpr.Push(ctx, resCh, d); err != nil {
				return err
			}
		}
		return nil
	})
	return resCh
}
