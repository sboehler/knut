package process

import (
	"context"
	"fmt"
	"time"

	"github.com/sboehler/knut/lib/common/amounts"
	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
	"github.com/sboehler/knut/lib/journal/val"
)

// Valuator produces valuated days.
type Valuator struct {
	Context   journal.Context
	Valuation *journal.Commodity

	values     amounts.Amounts
	amounts    amounts.Amounts
	normalized journal.NormalizedPrices
	date       time.Time
}

// ProcessStream computes prices.
func (pr *Valuator) ProcessStream(ctx context.Context, inCh <-chan *val.Day) (chan *val.Day, chan error) {
	errCh := make(chan error)
	resCh := make(chan *val.Day, 100)

	values := make(amounts.Amounts)

	go func() {
		defer close(resCh)
		defer close(errCh)

		for {
			day, ok, err := cpr.Pop(ctx, inCh)
			if !ok || err != nil {
				return
			}
			day.Values = values

			for _, t := range day.Day.Transactions {
				if errors := pr.valuateAndBookTransaction(ctx, day, t); cpr.Push(ctx, errCh, errors...) != nil {
					return
				}
			}

			pr.computeValuationTransactions(day)
			values = values.Clone()

			if cpr.Push(ctx, resCh, day) != nil {
				return
			}
		}
	}()

	return resCh, errCh
}

func (pr Valuator) valuateAndBookTransaction(ctx context.Context, day *val.Day, t *ast.Transaction) []error {
	var errors []error
	tx := t.Clone()
	for i, posting := range t.Postings {
		if pr.Valuation != nil && pr.Valuation != posting.Commodity {
			var err error
			if posting.Amount, err = day.Prices.Valuate(posting.Commodity, posting.Amount); err != nil {
				errors = append(errors, err)
			}
		}
		day.Values.Book(posting.Credit, posting.Debit, posting.Amount, posting.Commodity)
		tx.Postings[i] = posting
	}
	day.Transactions = append(day.Transactions, tx)
	return errors
}

// computeValuationTransactions checks whether the valuation for the positions
// corresponds to the amounts. If not, the difference is due to a valuation
// change of the previous amount, and a transaction is created to adjust the
// valuation.
func (pr Valuator) computeValuationTransactions(day *val.Day) {
	if pr.Valuation == nil {
		return
	}
	for pos, va := range day.Day.Amounts {
		if pos.Commodity == pr.Valuation {
			continue
		}
		var at = pos.Account.Type()
		if at != journal.ASSETS && at != journal.LIABILITIES {
			continue
		}
		value, err := day.Prices.Valuate(pos.Commodity, va)
		if err != nil {
			panic(fmt.Sprintf("no valuation found for commodity %s", pos.Commodity))
		}
		diff := value.Sub(day.Values[pos])
		if diff.IsZero() {
			continue
		}
		if !diff.IsZero() {
			credit := pr.Context.ValuationAccountFor(pos.Account)
			day.Transactions = append(day.Transactions, &ast.Transaction{
				Date:        day.Date,
				Description: fmt.Sprintf("Adjust value of %v in account %v", pos.Commodity, pos.Account),
				Postings: []ast.Posting{
					ast.NewPostingWithTargets(credit, pos.Account, pos.Commodity, diff, []*journal.Commodity{pos.Commodity}),
				},
			})
			day.Values.Book(credit, pos.Account, diff, pos.Commodity)
		}
	}
}

// Process valuates the transactions and inserts valuation transactions.
func (pr *Valuator) Process(ctx context.Context, d ast.Dated, ok bool, next func(ast.Dated) bool) error {
	if !ok {
		return nil
	}
	if pr.values == nil {
		pr.values = make(amounts.Amounts)
	}

	if pr.date != d.Date {
		// insert valuation transactions
		for pos, va := range pr.amounts {
			if pos.Commodity == pr.Valuation {
				continue
			}
			at := pos.Account.Type()
			if at != journal.ASSETS && at != journal.LIABILITIES {
				continue
			}
			value, err := pr.normalized.Valuate(pos.Commodity, va)
			if err != nil {
				return fmt.Errorf("no valuation found for commodity %s", pos.Commodity)
			}
			diff := value.Sub(pr.values[pos])
			if diff.IsZero() {
				continue
			}
			if !diff.IsZero() {
				credit := pr.Context.ValuationAccountFor(pos.Account)
				t := &ast.Transaction{
					Date:        pr.date,
					Description: fmt.Sprintf("Adjust value of %v in account %v", pos.Commodity, pos.Account),
					Postings: []ast.Posting{
						ast.NewPostingWithTargets(credit, pos.Account, pos.Commodity, diff, []*journal.Commodity{pos.Commodity}),
					},
				}
				pr.values.Book(credit, pos.Account, diff, pos.Commodity)
				if !next(ast.Dated{Date: pr.date, Elem: t}) {
					return nil
				}
			}
		}
		// replace amounts with values
		next(ast.Dated{Date: pr.date, Elem: pr.values.Clone()})
		pr.date = d.Date
	}
	switch dd := d.Elem.(type) {

	case journal.NormalizedPrices:
		pr.normalized = dd

	case *ast.Transaction:
		// valuate transaction
		for i, posting := range dd.Postings {
			if pr.Valuation != nil && pr.Valuation != posting.Commodity {
				var err error
				if posting.Amount, err = pr.normalized.Valuate(posting.Commodity, posting.Amount); err != nil {
					return err
				}
			}
			pr.values.Book(posting.Credit, posting.Debit, posting.Amount, posting.Commodity)
			dd.Postings[i] = posting
		}
		next(d)

	case amounts.Amounts:
		pr.amounts = dd

	default:
		next(d)
	}
	return nil
}
