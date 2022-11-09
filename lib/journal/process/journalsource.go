package process

import (
	"context"
	"fmt"
	"time"

	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/common/filter"
	"github.com/sboehler/knut/lib/common/mapper"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/parser"
	"go.uber.org/multierr"
)

// JournalSource emits journal data in daily batches.
type JournalSource struct {
	Context journal.Context

	Path     string
	Expand   bool
	AutoLoad bool

	journal *journal.JournalBuilder
}

func (js *JournalSource) Load(ctx context.Context) error {
	js.journal = journal.New(js.Context)
	p := parser.RecursiveParser{
		Context: js.Context,
		File:    js.Path,
	}
	var errs error
	err := cpr.Consume(ctx, p.Parse(ctx), func(d any) error {
		switch t := d.(type) {

		case error:
			errs = multierr.Append(errs, t)

		case *journal.Open:
			js.journal.AddOpen(t)

		case *journal.Price:
			js.journal.AddPrice(t)

		case *journal.Transaction:
			if t.Accrual != nil {
				for _, ts := range t.Accrual.Expand(t) {
					js.journal.AddTransaction(ts)
				}
			} else {
				js.journal.AddTransaction(t)
			}

		case *journal.Assertion:
			js.journal.AddAssertion(t)

		case *journal.Value:
			js.journal.AddValue(t)

		case *journal.Close:
			js.journal.AddClose(t)

		default:
			errs = multierr.Append(errs, fmt.Errorf("unknown: %#v", t))
		}
		return nil
	})
	if err != nil {
		return err
	}
	return errs
}

func (js JournalSource) Min() time.Time {
	return js.journal.Min()
}

func (js JournalSource) Max() time.Time {
	return js.journal.Max()
}

func (js JournalSource) Source(ctx context.Context, outCh chan<- *journal.Day) error {
	if js.AutoLoad {
		if err := js.Load(ctx); err != nil {
			return err
		}
	}
	for _, d := range js.journal.SortedDays() {
		if err := cpr.Push(ctx, outCh, d); err != nil {
			return err
		}
	}
	return nil
}

func (js JournalSource) Aggregate(ctx context.Context, v *journal.Commodity, f filter.Filter[journal.Key], m mapper.Mapper[journal.Key], c Collection) error {
	aggregator := &Aggregator{
		Valuation:  v,
		Collection: c,

		Filter: f,
		Mapper: m,
	}
	s := cpr.Compose[*journal.Day](js, &Balancer{Context: js.Context})
	if v != nil {
		s = cpr.Compose[*journal.Day](s, &PriceUpdater{
			Valuation: v,
		})
		s = cpr.Compose[*journal.Day](s, &Valuator{
			Context:   js.Context,
			Valuation: v,
		})
	}
	return cpr.Connect[*journal.Day](s, aggregator).Process(ctx)
}
