package process

import (
	"context"
	"fmt"

	"github.com/sboehler/knut/lib/common/amounts"
	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
	"golang.org/x/sync/errgroup"
)

// Valuator produces valuated days.
type Valuator struct {
	Context   journal.Context
	Valuation *journal.Commodity
}

// Process computes prices.
func (pr *Valuator) Process(ctx context.Context, g *errgroup.Group, inCh <-chan *ast.Day) <-chan *ast.Day {
	resCh := make(chan *ast.Day, 100)

	g.Go(func() error {
		defer close(resCh)

		values := make(amounts.Amounts)
		for {
			day, ok, err := cpr.Pop(ctx, inCh)
			if err != nil {
				return err
			}
			if !ok {
				break
			}
			day.Value = values

			for _, t := range day.Transactions {
				for i, posting := range t.Postings {
					if pr.Valuation != nil && pr.Valuation != posting.Commodity {
						var err error
						if posting.Amount, err = day.Normalized.Valuate(posting.Commodity, posting.Amount); err != nil {
							return err
						}
					}
					values.Book(posting.Credit, posting.Debit, posting.Amount, posting.Commodity)
					t.Postings[i] = posting
				}
			}

			pr.computeValuationTransactions(day)
			values = values.Clone()

			if err := cpr.Push(ctx, resCh, day); err != nil {
				return err
			}
		}
		return nil
	})

	return resCh
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
					ast.NewPostingWithTargets(credit, pos.Account, pos.Commodity, diff, []*journal.Commodity{pos.Commodity}),
				},
			}
			d.Value.Book(credit, pos.Account, diff, pos.Commodity)
			d.Transactions = append(d.Transactions, t)
		}
	}
	return nil

}
