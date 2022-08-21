package process

import (
	"context"

	"github.com/sboehler/knut/lib/common/amounts"
	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
)

type Aggregator struct {
	Context   journal.Context
	Mappers   amounts.Mapper
	Valuation *journal.Commodity
	Value     bool

	Amounts amounts.Amounts
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
				kc = agg.Mappers(kc)
				agg.Amounts.Add(kc, amt.Neg())
				kd := amounts.Key{
					Date:      t.Date(),
					Account:   b.Debit,
					Commodity: b.Commodity,
					Valuation: agg.Valuation,
				}
				kd = agg.Mappers(kd)
				agg.Amounts.Add(kd, amt)
			}
		}
	}
	return nil
}
