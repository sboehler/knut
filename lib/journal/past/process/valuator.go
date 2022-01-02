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

	errCh chan error
	resCh chan *val.Day

	values past.Amounts
}

// errOrExit will pass the error to errCh if it is not nil. It will return
// false if the error was successfully passed or if the error is nil. It
// returns true if the error couldn't be passed because the context was canceled.
func (pr *Valuator) errOrExit(ctx context.Context, err error) bool {
	if err != nil {
		select {
		case <-ctx.Done():
			return true
		case pr.errCh <- err:
			return false
		}
	}
	return false
}

// ProcessStream computes prices.
func (pr *Valuator) ProcessStream(ctx context.Context, inCh <-chan *val.Day) (chan *val.Day, chan error) {
	pr.errCh = make(chan error)
	pr.resCh = make(chan *val.Day, 100)

	pr.values = make(past.Amounts)

	go func() {
		defer close(pr.resCh)
		defer close(pr.errCh)

		for day := range inCh {
			day.Values = pr.values

			for _, t := range day.Day.Transactions {
				tv, err := pr.valuateAndBookTransaction(day, t)
				if pr.errOrExit(ctx, err) {
					return
				}
				day.Transactions = append(day.Transactions, tv)
			}

			pr.computeValuationTransactions(day)
			pr.values = pr.values.Clone()

			select {
			case pr.resCh <- day:
			case <-ctx.Done():
				return
			}
		}
	}()

	return pr.resCh, pr.errCh
}

func (pr Valuator) valuateAndBookTransaction(b *val.Day, t *ast.Transaction) (*ast.Transaction, error) {
	var res = t.Clone()
	for i, posting := range t.Postings {
		if pr.Valuation != nil && pr.Valuation != posting.Commodity {
			value, err := b.Prices.Valuate(posting.Commodity, posting.Amount)
			if err != nil {
				return nil, Error{t, err.Error()}
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
		if !diff.IsZero() {
			b.Transactions = append(b.Transactions, &ast.Transaction{
				Date:        b.Date,
				Description: desc,
				Postings: []ast.Posting{
					ast.NewPosting(valAcc, pos.Account, pos.Commodity, diff),
				},
			})
			b.Values.Book(valAcc, pos.Account, diff, pos.Commodity)
		}
	}
}
