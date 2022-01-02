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

	errCh chan error
	resCh chan *val.Day
}

// ProcessStream does the filtering.
func (pf Differ) ProcessStream(ctx context.Context, inCh <-chan *val.Day) (<-chan *val.Day, <-chan error) {
	pf.errCh = make(chan error)
	if !pf.Diff {
		close(pf.errCh)
		return inCh, pf.errCh
	}
	pf.resCh = make(chan *val.Day, 100)

	go func() {
		defer close(pf.resCh)
		defer close(pf.errCh)

		var (
			v    past.Amounts
			init bool
		)

		for d := range inCh {
			if init {
				res := &val.Day{
					Date:   d.Date,
					Values: d.Values.Clone(),
				}
				res.Values.Minus(v)
				select {
				case pf.resCh <- res:
				case <-ctx.Done():
					return
				}
			}
			init = true
			v = d.Values
		}

	}()
	return pf.resCh, pf.errCh
}
