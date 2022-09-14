package process

import (
	"context"

	"github.com/sboehler/knut/lib/common/amounts"
	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/common/filter"
	"github.com/sboehler/knut/lib/common/mapper"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
	"github.com/shopspring/decimal"
)

type Collection interface {
	Insert(k amounts.Key, v decimal.Decimal)
}

type Aggregator struct {
	Mapper    mapper.Mapper[amounts.Key]
	Filter    filter.Filter[amounts.Key]
	Valuation *journal.Commodity
	Value     bool

	Collection Collection
}

func (agg *Aggregator) Sink(ctx context.Context, inCh <-chan *ast.Day) error {
	if agg.Filter == nil {
		agg.Filter = filter.AllowAll[amounts.Key]
	}
	if agg.Mapper == nil {
		agg.Mapper = mapper.Identity[amounts.Key]
	}
	return cpr.Consume(ctx, inCh, func(d *ast.Day) error {
		for _, t := range d.Transactions {
			for _, b := range t.Postings {
				amt := b.Amount
				if agg.Valuation != nil {
					amt = b.Value
				}
				kc := amounts.Key{
					Date:      t.Date,
					Account:   b.Credit,
					Other:     b.Debit,
					Commodity: b.Commodity,
					Valuation: agg.Valuation,
				}
				if agg.Filter(kc) {
					kc = agg.Mapper(kc)
					agg.Collection.Insert(kc, amt.Neg())
				}
				kd := amounts.Key{
					Date:      t.Date,
					Account:   b.Debit,
					Other:     b.Credit,
					Commodity: b.Commodity,
					Valuation: agg.Valuation,
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
