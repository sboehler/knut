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
	for {
		day, ok, err := cpr.Pop(ctx, inCh)
		if err != nil {
			return err
		}
		if !ok {
			break
		}
		if pr.Valuation != nil {
			day.Value = values
			if err := pr.valuateTransactions(day); err != nil {
				return err
			}
			if err := pr.computeValuationTransactions(day); err != nil {
				return err
			}
			values = values.Clone()
		}
		if err := cpr.Push(ctx, outCh, day); err != nil {
			return err
		}
	}
	return nil
}

func (pr Valuator) valuateTransactions(d *ast.Day) error {
	var err error
	for _, t := range d.Transactions {
		for i := range t.Postings {
			posting := &t.Postings[i]
			if pr.Valuation != posting.Commodity {
				if posting.Value, err = d.Normalized.Valuate(posting.Commodity, posting.Amount); err != nil {
					return err
				}
			} else {
				posting.Value = posting.Amount
			}
			d.Value.Book(posting.Credit, posting.Debit, posting.Value, posting.Commodity)
		}
	}
	return nil
}

func (pr *Valuator) computeValuationTransactions(d *ast.Day) error {
	for pos, va := range d.Amounts {
		if pos.Commodity == pr.Valuation {
			continue
		}
		at := pos.Account.Type()
		if at != journal.ASSETS && at != journal.LIABILITIES {
			continue
		}
		value, err := d.Normalized.Valuate(pos.Commodity, va)
		if err != nil {
			return fmt.Errorf("no valuation found for commodity %s", pos.Commodity)
		}
		diff := value.Sub(d.Value[pos])
		if diff.IsZero() {
			continue
		}
		if !diff.IsZero() {
			credit := pr.Context.ValuationAccountFor(pos.Account)
			t := &ast.Transaction{
				Date:        d.Date,
				Description: fmt.Sprintf("Adjust value of %v in account %v", pos.Commodity, pos.Account),
				Postings: []ast.Posting{
					ast.NewValuePosting(credit, pos.Account, pos.Commodity, diff, []*journal.Commodity{pos.Commodity}),
				},
			}
			d.Value.Book(credit, pos.Account, diff, pos.Commodity)
			d.Transactions = append(d.Transactions, t)
		}
	}
	return nil

}
