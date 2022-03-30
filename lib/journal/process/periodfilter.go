package process

import (
	"context"
	"time"

	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/common/date"
	"github.com/sboehler/knut/lib/journal/ast"
	"golang.org/x/sync/errgroup"
)

// PeriodFilter filters the incoming days according to the dates
// specified.
type PeriodFilter struct {
	From, To time.Time
	Interval date.Interval
	Last     int
}

// Process does the filtering.
func (pf PeriodFilter) Process(ctx context.Context, g *errgroup.Group, inCh <-chan *ast.Day) <-chan *ast.Day {
	resCh := make(chan *ast.Day, 100)

	g.Go(func() error {

		defer close(resCh)

		var (
			periods []date.Period
			prev    = new(ast.Day)
			trx     []*ast.Transaction
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
					Date:         periods[index].End,
					Value:        prev.Value,
					Transactions: trx,
					Normalized:   prev.Normalized,
				}
				if err := cpr.Push(ctx, resCh, r); err != nil {
					return err
				}
				trx = nil
				index++
			}
			if !ok {
				break
			}
			trx = append(trx, day.Transactions...)
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
