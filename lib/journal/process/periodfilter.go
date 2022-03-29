package process

import (
	"context"
	"time"

	"github.com/sboehler/knut/lib/common/amounts"
	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/common/date"
	"github.com/sboehler/knut/lib/journal/ast"
	"github.com/sboehler/knut/lib/journal/val"
	"golang.org/x/sync/errgroup"
)

// PeriodFilter filters the incoming days according to the dates
// specified.
type PeriodFilter struct {
	From, To time.Time
	Interval date.Interval
	Last     int

	periods []date.Period
	index   int
	date    time.Time
	values  amounts.Amounts
}

// ProcessStream does the filtering.
func (pf PeriodFilter) ProcessStream(ctx context.Context, inCh <-chan *val.Day) (<-chan *val.Day, <-chan error) {
	resCh := make(chan *val.Day, 100)
	errCh := make(chan error)

	go func() {

		defer close(resCh)
		defer close(errCh)

		var (
			periods []date.Period
			prev    = new(val.Day)
			trx     []*ast.Transaction
			index   int
			init    bool
		)
		for {
			day, ok, err := cpr.Pop(ctx, inCh)
			if err != nil {
				return
			}
			if ok && !init {
				if len(day.Transactions) == 0 {
					continue
				}
				periods = pf.computeDates(day.Date)
				init = true
			}
			for index < len(periods) && (!ok || periods[index].End.Before(day.Date)) {
				r := &val.Day{
					Date:         periods[index].End,
					Values:       prev.Values,
					Prices:       prev.Prices,
					Transactions: trx,
				}
				if cpr.Push(ctx, resCh, r) != nil {
					return
				}
				trx = nil
				index++
			}
			if !ok {
				return
			}
			trx = append(trx, day.Transactions...)
			prev = day
		}
	}()
	return resCh, errCh

}

// Process2 does the filtering.
func (pf PeriodFilter) Process2(ctx context.Context, g *errgroup.Group, inCh <-chan *ast.Day) <-chan *ast.Day {
	resCh := make(chan *ast.Day, 100)

	g.Go(func() error {

		defer close(resCh)

		var (
			periods []date.Period
			prev    = new(ast.Day)
			index   int
			init    bool
		)
		for {
			day, ok, err := cpr.Pop(ctx, inCh)
			if err != nil {
				return err
			}
			if ok && !init {
				if len(day.Transactions) == 0 {
					continue
				}
				periods = pf.computeDates(day.Date)
				init = true
			}
			for index < len(periods) && (!ok || periods[index].End.Before(day.Date)) {
				r := &ast.Day{
					Date:       periods[index].End,
					Value:      prev.Value,
					Normalized: prev.Normalized,
				}
				if err := cpr.Push(ctx, resCh, r); err != nil {
					return err
				}
				index++
			}
			if !ok {
				break
			}
			prev = day
		}
		return nil
	})
	return resCh

}

func (pf *PeriodFilter) computeDates(t time.Time) []date.Period {
	from := pf.From
	if from.Before(t) {
		from = t
	}
	if pf.To.IsZero() {
		pf.To = date.Today()
	}
	dates := date.Periods(from, pf.To, pf.Interval)

	if pf.Last > 0 {
		last := pf.Last
		if len(dates) < last {
			last = len(dates)
		}
		if len(dates) > pf.Last {
			dates = dates[len(dates)-last:]
		}
	}
	return dates
}

// Process filters values according to the period.
func (pf *PeriodFilter) Process(ctx context.Context, d ast.Dated, next func(ast.Dated) bool) error {
	if pf.date.IsZero() {
		if _, ok := d.Elem.(*ast.Transaction); !ok {
			return nil
		}
		pf.date = d.Date
		pf.periods = pf.computeDates(d.Date)
	}

	for pf.index < len(pf.periods) && (pf.periods[pf.index].End.Before(d.Date)) {
		if !next(ast.Dated{Date: pf.periods[pf.index].End, Elem: pf.values}) {
			return nil
		}
		pf.index++
	}
	if v, ok := d.Elem.(amounts.Amounts); ok {
		pf.values = v
	}
	return nil
}

// Finalize implements Finalizer.
func (pf *PeriodFilter) Finalize(ctx context.Context, next func(ast.Dated) bool) error {
	for pf.index < len(pf.periods) {
		if !next(ast.Dated{Date: pf.periods[pf.index].End, Elem: pf.values}) {
			return nil
		}
		pf.index++
	}
	return nil
}
