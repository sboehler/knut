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
			days    []*ast.Day
			current int
			init    bool
			latest  *ast.Day
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
				latest = day
				init = true
			}
			for ; current < len(periods) && (!ok || periods[current].End.Before(day.Date)); current++ {
				pd := &ast.Day{
					Date:       periods[current].End,
					PeriodDays: days,
					Amounts:    latest.Amounts,
					Value:      latest.Value,
					Normalized: latest.Normalized,
				}
				if err := cpr.Push(ctx, resCh, pd); err != nil {
					return err
				}
				days = nil
			}
			if !ok {
				break
			}
			if current < len(periods) && periods[current].Contains(day.Date) {
				days = append(days, day)
			}
			latest = day
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
