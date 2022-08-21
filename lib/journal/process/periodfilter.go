package process

import (
	"context"
	"time"

	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/common/date"
	"github.com/sboehler/knut/lib/journal/ast"
)

// PeriodFilter filters the incoming days according to the dates
// specified.
type PeriodFilter struct {
	From, To time.Time
	Interval date.Interval
	Last     int
}

// Process does the filtering.
func (pf *PeriodFilter) Process(ctx context.Context, inCh <-chan *ast.Day, outCh chan<- *ast.Period) error {
	var (
		periods                 = date.PeriodsN(pf.From, pf.To, pf.Interval, pf.Last)
		previous, current, next *ast.Day
		ok                      bool
		err                     error
	)

	// previous is the last day before the start of the current period.
	// current is the last day before the end of the current period.
	previous, current, next = new(ast.Day), new(ast.Day), new(ast.Day)

	for _, period := range periods {
		var days []*ast.Day
		for next.Date.Before(period.Start) {
			previous = next
			current = next
			next, ok, err = cpr.Pop(ctx, inCh)
			if err != nil || !ok {
				return err
			}
		}
		for period.Contains(next.Date) {
			current = next
			days = append(days, current)
			next, ok, err = cpr.Pop(ctx, inCh)
			if err != nil {
				return err
			}
			if !ok {
				break
			}
		}
		res := &ast.Period{
			Period:      period,
			Days:        days,
			Amounts:     current.Amounts,
			Values:      current.Value,
			PrevAmounts: previous.Amounts,
			PrevValues:  previous.Value,
		}
		if err := cpr.Push(ctx, outCh, res); err != nil {
			return err
		}
		previous = current
	}
	// consume the remaining days
	for {
		if _, ok, err = cpr.Pop(ctx, inCh); err != nil || !ok {
			return err
		}
	}
}
