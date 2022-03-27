package process

import (
	"context"
	"time"

	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
	"github.com/sboehler/knut/lib/journal/past"
	"github.com/sboehler/knut/lib/journal/val"
)

// PriceUpdater updates the prices in a stream of days.
type PriceUpdater struct {
	Context   journal.Context
	Valuation *journal.Commodity

	date   time.Time
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

var _ ast.Processor = (*PriceUpdater)(nil)

// Process generates normalized prices.
func (pu *PriceUpdater) Process(ctx context.Context, d ast.Dated, ok bool, next func(ast.Dated) bool) error {
	if pu.Valuation == nil {
		next(d)
		return nil
	}
	if !ok {
		next(ast.Dated{Date: pu.date, Elem: pu.prices.Normalize(pu.Valuation)})
		return nil
	}
	if pu.prices == nil {
		pu.prices = make(journal.Prices)
	}

	switch p := d.Elem.(type) {
	case *ast.Price:
		if !pu.date.Equal(d.Date) {
			if !pu.date.IsZero() {
				next(ast.Dated{Date: d.Date, Elem: pu.prices.Normalize(pu.Valuation)})
			}
			pu.date = d.Date
		}
		pu.prices.Insert(p.Commodity, p.Price, p.Target)
	default:
		if !pu.date.IsZero() {
			next(ast.Dated{Date: d.Date, Elem: pu.prices.Normalize(pu.Valuation)})
			pu.date = time.Time{}
		}
		next(d)
	}
	return nil
}
