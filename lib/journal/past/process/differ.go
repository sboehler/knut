package process

import (
	"context"

	"github.com/sboehler/knut/lib/journal/past"
	"github.com/sboehler/knut/lib/journal/val"
)

// Differ filters the incoming days according to the dates
// specified.
type Differ struct {
	Diff bool
}

// ProcessStream does the filtering.
func (pf Differ) ProcessStream(ctx context.Context, inCh <-chan *val.Day) (<-chan *val.Day, <-chan error) {
	errCh := make(chan error)
	if !pf.Diff {
		close(errCh)
		return inCh, errCh
	}
	resCh := make(chan *val.Day, 100)

	go func() {
		defer close(resCh)
		defer close(errCh)

		var v past.Amounts

		for d := range inCh {
			if v != nil {
				res := &val.Day{
					Date:   d.Date,
					Values: d.Values.Clone(),
				}
				res.Values.Minus(v)
				select {
				case resCh <- res:
				case <-ctx.Done():
					return
				}
			}
			v = d.Values
		}

	}()
	return resCh, errCh
}
