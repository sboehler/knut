package process

import (
	"context"
	"sort"
	"time"

	"github.com/sboehler/knut/lib/common/amounts"
	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/common/date"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
)

type Aggregator struct {
	Context journal.Context

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

	Amounts amounts.Amounts

	tp timePartition
}

func (agg *Aggregator) Sink(ctx context.Context, inCh <-chan *ast.Day) error {
	agg.Amounts = make(amounts.Amounts)
	for {
		d, ok, err := cpr.Pop(ctx, inCh)
		if err != nil {
			return err
		}
		if !ok {
			break
		}
		if agg.tp == nil {
			if len(d.Transactions) == 0 {
				continue
			}
			if agg.From.IsZero() {
				agg.From = d.Date
			}
			agg.tp = createPartition(agg.From, agg.To, agg.Interval, agg.Last)
		}
		dt := agg.tp.shard(d.Date)
		for _, t := range d.Transactions {
			for _, b := range t.Postings() {
				kc := amounts.Key{
					Date:      dt,
					Account:   agg.Context.Accounts().Map(b.Credit, agg.Mapping),
					Commodity: b.Commodity,
				}
				agg.Amounts[kc] = agg.Amounts[kc].Sub(b.Amount)
				kd := amounts.Key{
					Date:      dt,
					Account:   agg.Context.Accounts().Map(b.Debit, agg.Mapping),
					Commodity: b.Commodity,
				}
				agg.Amounts[kd] = agg.Amounts[kd].Add(b.Amount)
			}
			if agg.Valuation != nil {
				for _, b := range t.Postings() {
					kc := amounts.Key{
						Date:      dt,
						Account:   agg.Context.Accounts().Map(b.Credit, agg.Mapping),
						Commodity: b.Commodity,
						Valuation: agg.Valuation,
					}
					agg.Amounts[kc] = agg.Amounts[kc].Sub(b.Value)
					kd := amounts.Key{
						Date:      dt,
						Account:   agg.Context.Accounts().Map(b.Debit, agg.Mapping),
						Commodity: b.Commodity,
						Valuation: agg.Valuation,
					}
					agg.Amounts[kd] = agg.Amounts[kd].Add(b.Value)
				}
			}
		}
	}
	return nil
}

type timePartition []time.Time

func (tp timePartition) Transform(k amounts.Key) amounts.Key {
	k.Date = tp.shard(k.Date)
	return k
}

func (tp timePartition) shard(t time.Time) time.Time {
	index := sort.Search(len(tp), func(i int) bool {
		return !tp[i].Before(t)
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
	if n > 0 && len(res) > n {
		res = res[len(res)-n:]
	}
	return res
}
