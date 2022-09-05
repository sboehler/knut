package process

import (
	"context"

	"github.com/sboehler/knut/lib/common/amounts"
	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/common/filter"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
	"github.com/shopspring/decimal"
)

type Collection interface {
	Insert(k amounts.Key, v decimal.Decimal)
}

type Aggregator struct {
	Mappers   amounts.Mapper
	Filter    filter.Filter[amounts.Key]
	Valuation *journal.Commodity
	Value     bool

	Collection Collection
}

func (agg *Aggregator) Sink(ctx context.Context, inCh <-chan *ast.Day) error {
	if agg.Filter == nil {
		agg.Filter = filter.Default[amounts.Key]
	}
	if agg.Mappers == nil {
		agg.Mappers = amounts.DefaultMapper
	}
	for {
		d, ok, err := cpr.Pop(ctx, inCh)
		if err != nil {
			return err
		}
		if !ok {
			break
		}
		for _, t := range d.Transactions {
			for _, b := range t.Postings() {
				amt := b.Amount
				if agg.Valuation != nil {
					amt = b.Value
				}
				kc := amounts.Key{
					Date:      t.Date(),
					Account:   b.Credit,
					Commodity: b.Commodity,
					Valuation: agg.Valuation,
				}
				if agg.Filter(kc) {
					kc = agg.Mappers(kc)
					agg.Collection.Insert(kc, amt.Neg())
				}
				kd := amounts.Key{
					Date:      t.Date(),
					Account:   b.Debit,
					Commodity: b.Commodity,
					Valuation: agg.Valuation,
				}
				if agg.Filter(kd) {
					kd = agg.Mappers(kd)
					agg.Collection.Insert(kd, amt)
				}
			}
		}
	}
	return nil
}
