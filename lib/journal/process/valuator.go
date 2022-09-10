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
func (pr *Valuator) Process(ctx context.Context, inCh <-chan *ast.Day, outCh chan<- *ast.Day) error {
	values := make(amounts.Amounts)
	return cpr.Consume(ctx, inCh, func(d *ast.Day) error {
		if pr.Valuation != nil {
			d.Value = values
			if err := pr.valuateTransactions(d); err != nil {
				return err
			}
			if err := pr.computeValuationTransactions(d); err != nil {
				return err
			}
			values = values.Clone()
		}
		return cpr.Push(ctx, outCh, d)
	})
}

func (pr Valuator) valuateTransactions(d *ast.Day) error {
	var (
		err error
		res []*ast.Transaction
	)
	for _, t := range d.Transactions {
		tb := t.ToBuilder()
		for i := range tb.Postings {
			posting := &tb.Postings[i]
			if pr.Valuation != posting.Commodity {
				if posting.Value, err = d.Normalized.Valuate(posting.Commodity, posting.Amount); err != nil {
					return err
				}
			} else {
				posting.Value = posting.Amount
			}
			d.Value.Add(amounts.AccountCommodityKey(posting.Credit, posting.Commodity), posting.Value.Neg())
			d.Value.Add(amounts.AccountCommodityKey(posting.Debit, posting.Commodity), posting.Value)
		}
		res = append(res, tb.Build())
	}
	d.Transactions = res
	return nil
}

func (pr *Valuator) computeValuationTransactions(d *ast.Day) error {
	for pos, va := range d.Amounts {
		if pos.Commodity == pr.Valuation {
			continue
		}
		if !pos.Account.IsAL() {
			continue
		}
		value, err := d.Normalized.Valuate(pos.Commodity, va)
		if err != nil {
			return fmt.Errorf("no valuation found for commodity %s", pos.Commodity.Name())
		}
		diff := value.Sub(d.Value[pos])
		if diff.IsZero() {
			continue
		}
		credit := pr.Context.ValuationAccountFor(pos.Account)
		t := ast.TransactionBuilder{
			Date:        d.Date,
			Description: fmt.Sprintf("Adjust value of %s in account %s", pos.Commodity.Name(), pos.Account.Name()),
			Postings: []ast.Posting{
				ast.NewValuePosting(credit, pos.Account, pos.Commodity, diff, []*journal.Commodity{pos.Commodity}),
			},
		}.Build()
		d.Value.Add(amounts.AccountCommodityKey(credit, pos.Commodity), diff.Neg())
		d.Value.Add(amounts.AccountCommodityKey(pos.Account, pos.Commodity), diff)
		d.Transactions = append(d.Transactions, t)
	}
	return nil

}
