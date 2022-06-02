package process

import (
	"context"
	"sort"
	"time"

	"github.com/sboehler/knut/lib/common/amounts2"
	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/common/date"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
)

type Aggregator struct {
	// date sharding
	From, To time.Time
	Interval date.Interval
	Last     int

	// account sharding
	Mapping journal.Mapping

	// commodity sharding
	// TODO: add some commodity remapping (e.g. currency vs security)

	// value sharding
	Valuation *journal.Commodity

	Amounts amounts2.Amounts

	tp timePartition
}

func (rb *Aggregator) Sink(ctx context.Context, inCh <-chan *ast.Day) error {
	rb.Amounts = make(amounts2.Amounts)
	for {
		d, ok, err := cpr.Pop(ctx, inCh)
		if err != nil {
			return err
		}
		if !ok {
			break
		}
		if rb.tp == nil {
			if len(d.Transactions) == 0 {
				continue
			}
			if rb.From.IsZero() {
				rb.From = d.Date
			}
			rb.tp = createPartition(rb.From, rb.To, rb.Interval, rb.Last)
		}
		dt := rb.tp.shard(d.Date)
		for _, t := range d.Transactions {
			for _, b := range t.Postings() {
				kc := amounts2.Key{
					Date:      dt,
					Account:   b.Credit.Map(rb.Mapping),
					Commodity: b.Commodity,
				}
				rb.Amounts[kc] = rb.Amounts[kc].Sub(b.Amount)
				kd := amounts2.Key{
					Date:      dt,
					Account:   b.Debit.Map(rb.Mapping),
					Commodity: b.Commodity,
				}
				rb.Amounts[kd] = rb.Amounts[kd].Add(b.Amount)
			}
			if rb.Valuation != nil {
				for _, b := range t.Postings() {
					kc := amounts2.Key{
						Date:      dt,
						Account:   b.Credit.Map(rb.Mapping),
						Commodity: b.Commodity,
						Valuation: rb.Valuation,
					}
					rb.Amounts[kc] = rb.Amounts[kc].Sub(b.Value)
					kd := amounts2.Key{
						Date:      dt,
						Account:   b.Debit.Map(rb.Mapping),
						Commodity: b.Commodity,
						Valuation: rb.Valuation,
					}
					rb.Amounts[kd] = rb.Amounts[kd].Add(b.Value)
				}
			}
		}
	}
	return nil
}

type timePartition []time.Time

func (tp timePartition) shard(t time.Time) time.Time {
	index := sort.Search(len(tp), func(i int) bool {
		return tp[i].After(t)
	})
	if index == len(tp) {
		return time.Time{}
	}
	return tp[index]
}

func createPartition(t0, t1 time.Time, p date.Interval, n int) timePartition {
	var res []time.Time
	if p == date.Once {
		if t0.Before(t1) {
			res = append(res, t1)
		}
	} else {
		for t := t0; !t.After(t1); t = date.EndOf(t, p).AddDate(0, 0, 1) {
			ed := date.EndOf(t, p)
			if ed.After(t1) {
				ed = t1
			}
			res = append(res, ed)
		}
	}
	if len(res) > n {
		res = res[len(res)-n:]
	}
	return res
}
