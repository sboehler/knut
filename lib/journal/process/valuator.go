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

// Process valuates transactions.
func (val Valuator) Process(ctx context.Context, inCh <-chan *ast.Day, outCh chan<- *ast.Day) error {
	values := make(amounts.Amounts)
	return cpr.Consume(ctx, inCh, func(d *ast.Day) error {
		if val.Valuation != nil {
			if err := val.valuateTransactions(d, values); err != nil {
				return err
			}
			if err := val.valuateGains(d, values); err != nil {
				return err
			}
			d.Value = values.Clone()
		}
		return cpr.Push(ctx, outCh, d)
	})
}

func (val Valuator) valuateTransactions(d *ast.Day, values amounts.Amounts) error {
	for _, t := range d.Transactions {
		for i := range t.Postings {
			posting := &t.Postings[i]
			v := posting.Amount
			var err error
			if val.Valuation != posting.Commodity {
				if v, err = d.Normalized.Valuate(posting.Commodity, posting.Amount); err != nil {
					return err
				}
			}
			posting.Value = v
			values.Add(amounts.AccountCommodityKey(posting.Credit, posting.Commodity), posting.Value.Neg())
			values.Add(amounts.AccountCommodityKey(posting.Debit, posting.Commodity), posting.Value)
		}

	}
	return nil
}

func (val Valuator) valuateGains(d *ast.Day, values amounts.Amounts) error {
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
		gain := value.Sub(values[pos])
		if gain.IsZero() {
			continue
		}
		credit := val.Context.ValuationAccountFor(pos.Account)
		d.Transactions = append(d.Transactions, ast.TransactionBuilder{
			Date:        d.Date,
			Description: fmt.Sprintf("Adjust value of %s in account %s", pos.Commodity.Name(), pos.Account.Name()),
			Postings: []ast.Posting{
				ast.NewValuePosting(credit, pos.Account, pos.Commodity, gain, []*journal.Commodity{pos.Commodity}),
			},
		}.Build())
		values.Add(pos, gain)
		values.Add(amounts.AccountCommodityKey(credit, pos.Commodity), gain.Neg())
	}
	return nil

}
