package process

import (
	"context"

	"github.com/sboehler/knut/lib/common/amounts"
	"github.com/sboehler/knut/lib/common/cpr"
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

		var (
			v    amounts.Amounts
			init bool
		)

		for {
			d, ok, err := cpr.Pop(ctx, inCh)
			if !ok || err != nil {
				return
			}
			if init {
				res := &val.Day{
					Date:   d.Date,
					Values: d.Values.Clone().Minus(v),
				}
				if cpr.Push(ctx, resCh, res) != nil {
					return
				}
			}
			init = true
			v = d.Values
		}

	}()
	return resCh, errCh
}
