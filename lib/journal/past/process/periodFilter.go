package process

import (
	"context"
	"time"

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
	var (
		resCh = make(chan *val.Day, 100)
		errCh = make(chan error)
		index int
	)

	go func() {
		defer close(resCh)
		defer close(errCh)
		if pf.To.IsZero() {
			pf.To = date.Today()
		}
		var dates []time.Time
		select {
		case day, ok := <-inCh:
			if !ok {
				return
			}
			if pf.From.Before(day.Date) {
				pf.From = day.Date
			}
			if !pf.From.Before(pf.To) {
				return
			}
			dates = date.Series(pf.From, pf.To, pf.Period)

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
			for index < len(dates) && dates[index].Before(day.Date) {
				select {
				case resCh <- &val.Day{Date: dates[index]}:
					index++
				case <-ctx.Done():
					return
				}
			}
			if index < len(dates) && dates[index].Equal(day.Date) {
				select {
				case resCh <- &val.Day{
					Date:         dates[index],
					Values:       day.Values,
					Prices:       day.Prices,
					Transactions: day.Transactions,
				}:
					index++
				case <-ctx.Done():
					return
				}
			}

		case <-ctx.Done():
			return
		}

		var (
			day *val.Day
			ok  bool
		)
		for inCh != nil {
			select {
			case day, ok = <-inCh:
				if !ok {
					inCh = nil
					break
				}
				for index < len(dates) && !dates[index].After(day.Date) {
					r := &val.Day{
						Date:         dates[index],
						Values:       day.Values,
						Prices:       day.Prices,
						Transactions: day.Transactions,
					}
					select {
					case resCh <- r:
						index++
					case <-ctx.Done():
						return
					}
				}
			case <-ctx.Done():
				return
			}
		}
		for index < len(dates) {
			r := &val.Day{
				Date:         dates[index],
				Values:       day.Values,
				Prices:       day.Prices,
				Transactions: day.Transactions,
			}
			select {
			case resCh <- r:
				index++
			case <-ctx.Done():
				return
			}
		}
	}()

	return resCh, errCh

}
