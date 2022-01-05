package process

import (
	"context"
	"time"

	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/common/date"
	"github.com/sboehler/knut/lib/journal/val"
)

// PeriodFilter filters the incoming days according to the dates
// specified.
type PeriodFilter struct {
	From, To time.Time
	Period   date.Period
	Last     int
	Diff     bool
}

// ProcessStream does the filtering.
func (pf PeriodFilter) ProcessStream(ctx context.Context, inCh <-chan *val.Day) (<-chan *val.Day, <-chan error) {
	resCh := make(chan *val.Day, 100)
	errCh := make(chan error)

	var index int

	go func() {

		defer close(resCh)
		defer close(errCh)

		if pf.To.IsZero() {
			pf.To = date.Today()
		}
		var (
			dates []time.Time
			init  bool
			day   *val.Day
		)
		for {
			if d, ok, err := cpr.Pop(ctx, inCh); err != nil {
				return
			} else if !ok {
				break
			} else {
				day = d
			}
			if !init {
				if len(day.Transactions) == 0 {
					continue
				}
				dates = pf.computeDates(day)
				for index < len(dates) && dates[index].Before(day.Date) {
					r := &val.Day{
						Date: dates[index],
					}
					if cpr.Push(ctx, resCh, r) != nil {
						return
					}
					index++
				}
				init = true
			}

			for index < len(dates) && !dates[index].After(day.Date) {
				r := &val.Day{
					Date:   dates[index],
					Values: day.Values,
					Prices: day.Prices,
					//Transactions: day.Transactions,
				}
				if cpr.Push(ctx, resCh, r) != nil {
					return
				}
				index++
			}
		}
		for index < len(dates) {
			r := &val.Day{
				Date:   dates[index],
				Values: day.Values,
				Prices: day.Prices,
				//Transactions: day.Transactions,
			}
			if cpr.Push(ctx, resCh, r) != nil {
				return
			}
			index++
		}
	}()
	return resCh, errCh

}

func (pf *PeriodFilter) computeDates(day *val.Day) []time.Time {
	from := pf.From
	if pf.From.Before(day.Date) {
		from = day.Date
	}
	if !from.Before(pf.To) {
		return nil
	}
	dates := date.Series(from, pf.To, pf.Period)

	if pf.Last > 0 {
		last := pf.Last
		if len(dates) < last {
			last = len(dates)
		}
		if pf.Diff {
			last++
		}
		if len(dates) > pf.Last {
			dates = dates[len(dates)-last:]
		}
	}
	return dates
}
