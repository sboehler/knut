package process

import (
	"context"
	"fmt"

	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
	"github.com/sboehler/knut/lib/journal/past"
	"github.com/sboehler/knut/lib/journal/val"
)

// Valuator produces valuated days.
type Valuator struct {
	Context   journal.Context
	Valuation *journal.Commodity
}

// ProcessStream computes prices.
func (pr Valuator) ProcessStream(ctx context.Context, inCh <-chan *val.Day) (chan *val.Day, chan error) {

	var (
		errCh = make(chan error)
		resCh = make(chan *val.Day, 100)

		errOrExit = func(err error) bool {
			select {
			case errCh <- err:
				return false
			case <-ctx.Done():
				return true
			}
		}
	)
	go func() {
		defer close(resCh)
		defer close(errCh)

		var values = make(past.Amounts)

		for day := range inCh {
			day.Values = values

			for _, t := range day.Day.Transactions {
				tv, err := pr.valuateAndBookTransaction(day, t)
				if err != nil {
					errOrExit(err)
					return
				}
				day.Transactions = append(day.Transactions, tv)
			}

			pr.computeValuationTransactions(day)

			values = values.Clone()
			select {
			case resCh <- day:
			case <-ctx.Done():
				return
			}
		}
	}()

	return resCh, errCh
}

func (pr Valuator) valuateAndBookTransaction(b *val.Day, t *ast.Transaction) (*ast.Transaction, error) {
	var res = t.Clone()
	for i, posting := range t.Postings {
		if pr.Valuation != nil && pr.Valuation != posting.Commodity {
			value, err := b.Prices.Valuate(posting.Commodity, posting.Amount)
			if err != nil {
				return nil, Error{t, fmt.Sprintf("no price found for commodity %s", posting.Commodity)}
			}
			posting.Amount = value
		}
		b.Values.Book(posting.Credit, posting.Debit, posting.Amount, posting.Commodity)
		res.Postings[i] = posting
	}
	return res, nil
}

// computeValuationTransactions checks whether the valuation for the positions
// corresponds to the amounts. If not, the difference is due to a valuation
// change of the previous amount, and a transaction is created to adjust the
// valuation.
func (pr Valuator) computeValuationTransactions(b *val.Day) {
	if pr.Valuation == nil {
		return
	}
	for pos, va := range b.Day.Amounts {
		if pos.Commodity == pr.Valuation {
			continue
		}
		var at = pos.Account.Type()
		if at != journal.ASSETS && at != journal.LIABILITIES {
			continue
		}
		value, err := b.Prices.Valuate(pos.Commodity, va)
		if err != nil {
			panic(fmt.Sprintf("no valuation found for commodity %s", pos.Commodity))
		}
		var diff = value.Sub(b.Values[pos])
		if diff.IsZero() {
			continue
		}
		valAcc, err := pr.Context.ValuationAccountFor(pos.Account)
		if err != nil {
			panic(fmt.Sprintf("could not obtain valuation account for account %s", pos.Account))
		}
		desc := fmt.Sprintf("Adjust value of %v in account %v", pos.Commodity, pos.Account)
		if diff.IsPositive() {

			// create a transaction to adjust the valuation
			b.Transactions = append(b.Transactions, &ast.Transaction{
				Date:        b.Date,
				Description: desc,
				Postings: []ast.Posting{
					{
						Credit:    valAcc,
						Debit:     pos.Account,
						Amount:    diff,
						Commodity: pos.Commodity,
					},
				},
			})
			b.Values.Book(valAcc, pos.Account, diff, pos.Commodity)
		} else {

			// create a transaction to adjust the valuation
			b.Transactions = append(b.Transactions, &ast.Transaction{
				Date:        b.Date,
				Description: desc,
				Postings: []ast.Posting{
					{
						Credit:    pos.Account,
						Debit:     valAcc,
						Amount:    diff.Neg(),
						Commodity: pos.Commodity,
					},
				},
			})
			b.Values.Book(pos.Account, valAcc, diff.Neg(), pos.Commodity)
		}
	}
}
