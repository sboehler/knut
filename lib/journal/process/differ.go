package process

import (
	"context"

	"github.com/sboehler/knut/lib/common/amounts"
	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
	"golang.org/x/sync/errgroup"
)

// PeriodDiffer filters the incoming days according to the dates
// specified.
type PeriodDiffer struct {
	Valuation *journal.Commodity
}

// Process does the diffing.
func (pf PeriodDiffer) Process(ctx context.Context, g *errgroup.Group, inCh <-chan *ast.Day) <-chan *ast.Day {
	resCh := make(chan *ast.Day, 100)

	g.Go(func() error {
		defer close(resCh)

		grp, ctx := errgroup.WithContext(ctx)
		for {
			d, ok, err := cpr.Pop(ctx, inCh)
			if err != nil {
				return err
			}
			if !ok {
				break
			}
			grp.Go(func() error {
				amts := make(amounts.Amounts)
				value := make(amounts.Amounts)
				for _, pd := range d.PeriodDays {
					for _, trx := range pd.Transactions {
						for _, p := range trx.Postings {
							amts.Book(p.Credit, p.Debit, p.Amount, p.Commodity)
							if pf.Valuation != nil {
								value.Book(p.Credit, p.Debit, p.Value, p.Commodity)
							}
						}
					}
				}
				d.Amounts = amts
				if pf.Valuation != nil {
					d.Value = value
				}
				if err := cpr.Push(ctx, resCh, d); err != nil {
					return err
				}
				return nil
			})
		}
		return grp.Wait()
	})
	return resCh
}
