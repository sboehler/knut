package process

import (
	"context"

	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/common/filter"
	"github.com/sboehler/knut/lib/common/mapper"
	"github.com/sboehler/knut/lib/journal"
	"github.com/shopspring/decimal"
)

type Collection interface {
	Insert(k journal.Key, v decimal.Decimal)
}

type Aggregator struct {
	Mapper    mapper.Mapper[journal.Key]
	Filter    filter.Filter[journal.Key]
	Valuation *journal.Commodity

	Collection Collection
}

func (agg *Aggregator) Sink(ctx context.Context, inCh <-chan *journal.Day) error {
	if agg.Filter == nil {
		agg.Filter = filter.AllowAll[journal.Key]
	}
	if agg.Mapper == nil {
		agg.Mapper = mapper.Identity[journal.Key]
	}
	return cpr.Consume(ctx, inCh, func(d *journal.Day) error {
		for _, t := range d.Transactions {
			for _, b := range t.Postings {
				amt := b.Amount
				if agg.Valuation != nil {
					amt = b.Value
				}
				kc := journal.Key{
					Date:        t.Date,
					Account:     b.Credit,
					Other:       b.Debit,
					Commodity:   b.Commodity,
					Valuation:   agg.Valuation,
					Description: t.Description,
				}
				if agg.Filter(kc) {
					kc = agg.Mapper(kc)
					agg.Collection.Insert(kc, amt.Neg())
				}
				kd := journal.Key{
					Date:        t.Date,
					Account:     b.Debit,
					Other:       b.Credit,
					Commodity:   b.Commodity,
					Valuation:   agg.Valuation,
					Description: t.Description,
				}
				if agg.Filter(kd) {
					kd = agg.Mapper(kd)
					agg.Collection.Insert(kd, amt)
				}
			}
		}
		return nil
	})
}
