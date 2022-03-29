package process

import (
	"context"
	"time"

	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
	"github.com/sboehler/knut/lib/journal/past"
	"github.com/sboehler/knut/lib/journal/val"
	"golang.org/x/sync/errgroup"
)

// PriceUpdater updates the prices in a stream of days.
type PriceUpdater struct {
	Context   journal.Context
	Valuation *journal.Commodity

	date   time.Time
	send   bool
	prices journal.Prices
}

// ProcessStream computes prices.
func (pu PriceUpdater) ProcessStream(ctx context.Context, inCh <-chan *past.Day) (<-chan *val.Day, <-chan error) {
	var (
		resCh = make(chan *val.Day, 100)
		errCh = make(chan error)
		prc   = make(journal.Prices)
	)
	go func() {
		defer close(resCh)
		defer close(errCh)

		var previous *val.Day
		for {
			day, ok, err := cpr.Pop(ctx, inCh)
			if !ok || err != nil {
				return
			}
			var npr journal.NormalizedPrices
			if pu.Valuation != nil {
				if day.AST != nil && len(day.AST.Prices) > 0 {
					for _, p := range day.AST.Prices {
						prc.Insert(p.Commodity, p.Price, p.Target)
					}
					npr = prc.Normalize(pu.Valuation)
				} else if previous == nil {
					npr = prc.Normalize(pu.Valuation)
				} else {
					npr = previous.Prices
				}
			}
			vday := &val.Day{
				Date:   day.Date,
				Day:    day,
				Prices: npr,
			}
			previous = vday
			if cpr.Push(ctx, resCh, vday) != nil {
				return
			}
		}
	}()
	return resCh, errCh
}

// Process2 computes prices.
func (pu PriceUpdater) Process2(ctx context.Context, g *errgroup.Group, inCh <-chan *ast.Day) <-chan *ast.Day {
	var (
		resCh = make(chan *ast.Day, 100)
	)
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

var _ ast.Processor = (*PriceUpdater)(nil)

// Process generates normalized prices.
func (pu *PriceUpdater) Process(ctx context.Context, d ast.Dated, next func(ast.Dated) bool) error {
	if pu.Valuation == nil {
		next(d)
		return nil
	}
	if pu.prices == nil {
		pu.prices = make(journal.Prices)
	}

	switch p := d.Elem.(type) {
	case *ast.Price:
		if pu.send && !pu.date.Equal(d.Date) {
			if !next(ast.Dated{Date: pu.date, Elem: pu.prices.Normalize(pu.Valuation)}) {
				return nil
			}
		}
		pu.send = true
		pu.prices.Insert(p.Commodity, p.Price, p.Target)
	default:
		if pu.send {
			if !next(ast.Dated{Date: pu.date, Elem: pu.prices.Normalize(pu.Valuation)}) {
				return nil
			}
			pu.send = false
		}
		next(d)
	}
	pu.date = d.Date
	return nil
}

// Finalize implements Finalize.
func (pu *PriceUpdater) Finalize(ctx context.Context, next func(ast.Dated) bool) error {
	if pu.send {
		next(ast.Dated{Date: pu.date, Elem: pu.prices.Normalize(pu.Valuation)})
	}
	return nil
}
