package process

import (
	"context"
	"fmt"

	"github.com/sboehler/knut/lib/common/amounts"
	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
)

// Valuator produces valuated days.
type Valuator struct {
	Context   journal.Context
	Valuation *journal.Commodity
}

// Process computes prices.
func (val Valuator) Process(ctx context.Context, inCh <-chan *ast.Day, outCh chan<- *ast.Day) error {
	values := make(amounts.Amounts)
	return cpr.Consume(ctx, inCh, func(d *ast.Day) error {
		if val.Valuation != nil {
			if err := val.valuateTransactions(d, values); err != nil {
				return err
			}
			if err := val.computeValuationTransactions(d, values); err != nil {
				return err
			}
			d.Value = values.Clone()
		}
		return cpr.Push(ctx, outCh, d)
	})
}

func (val Valuator) valuateTransactions(d *ast.Day, values amounts.Amounts) error {
	var (
		err error
		res []*ast.Transaction
	)
	for _, t := range d.Transactions {
		tb := t.ToBuilder()
		for i := range tb.Postings {
			posting := &tb.Postings[i]
			if val.Valuation != posting.Commodity {
				if posting.Value, err = d.Normalized.Valuate(posting.Commodity, posting.Amount); err != nil {
					return err
				}
			} else {
				posting.Value = posting.Amount
			}
			values.Add(amounts.AccountCommodityKey(posting.Credit, posting.Commodity), posting.Value.Neg())
			values.Add(amounts.AccountCommodityKey(posting.Debit, posting.Commodity), posting.Value)
		}
		res = append(res, tb.Build())
	}
	d.Transactions = res
	return nil
}

func (val Valuator) computeValuationTransactions(d *ast.Day, values amounts.Amounts) error {
	for pos, amt := range d.Amounts {
		if pos.Commodity == val.Valuation {
			continue
		}
		if !pos.Account.IsAL() {
			continue
		}
		value, err := d.Normalized.Valuate(pos.Commodity, amt)
		if err != nil {
			return fmt.Errorf("no valuation found for commodity %s", pos.Commodity.Name())
		}
		diff := value.Sub(values[pos])
		if diff.IsZero() {
			continue
		}
		credit := val.Context.ValuationAccountFor(pos.Account)
		t := ast.TransactionBuilder{
			Date:        d.Date,
			Description: fmt.Sprintf("Adjust value of %s in account %s", pos.Commodity.Name(), pos.Account.Name()),
			Postings: []ast.Posting{
				ast.NewValuePosting(credit, pos.Account, pos.Commodity, diff, []*journal.Commodity{pos.Commodity}),
			},
		}.Build()
		values.Add(amounts.AccountCommodityKey(credit, pos.Commodity), diff.Neg())
		values.Add(amounts.AccountCommodityKey(pos.Account, pos.Commodity), diff)
		d.Transactions = append(d.Transactions, t)
	}
	return nil

}
