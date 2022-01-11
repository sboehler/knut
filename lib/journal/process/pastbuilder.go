package process

import (
	"context"
	"fmt"

	"github.com/sboehler/knut/lib/common/amounts"
	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
	"github.com/sboehler/knut/lib/journal/past"
)

// PASTBuilder processes ASTs.
type PASTBuilder struct {

	// The context of this journal.
	Context journal.Context
}

// ProcessAST processes an AST to a stream of past.Day. It check assertions
// and the usage of open and closed accounts. It will also
// resolve Value directives and convert them to transactions.
func (pr *PASTBuilder) ProcessAST(ctx context.Context, inCh <-chan *ast.AST) (<-chan *past.Day, <-chan error) {

	errCh := make(chan error)
	resCh := make(chan *past.Day, 100)

	go func() {
		defer close(resCh)
		defer close(errCh)

		for {
			a, ok, err := cpr.Pop(ctx, inCh)
			if !ok || err != nil {
				return
			}
			amounts := make(amounts.Amounts)
			accounts := make(accounts)

			for _, d := range a.SortedDays() {
				var (
					transactions []*ast.Transaction
					errors       []error
				)
				if errors = pr.processOpenings(ctx, accounts, d); cpr.Push(ctx, errCh, errors...) != nil {
					return
				}
				if errors = pr.processTransactions(ctx, accounts, amounts, d); cpr.Push(ctx, errCh, errors...) != nil {
					return
				}
				if transactions, errors = pr.processValues(ctx, accounts, amounts, d); cpr.Push(ctx, errCh, errors...) != nil {
					return
				}
				if errors = pr.processAssertions(ctx, accounts, amounts, d); cpr.Push(ctx, errCh, errors...) != nil {
					return
				}
				if errors = pr.processClosings(ctx, accounts, amounts, d); cpr.Push(ctx, errCh, errors...) != nil {
					return
				}
				res := &past.Day{
					Date:         d.Date,
					AST:          d,
					Transactions: append(transactions, d.Transactions...),
					Amounts:      amounts,
				}
				amounts = amounts.Clone()
				if cpr.Push(ctx, resCh, res) != nil {
					return
				}
			}
		}
	}()
	return resCh, errCh
}

func (pr *PASTBuilder) processOpenings(ctx context.Context, accounts accounts, d *ast.Day) []error {
	var errors []error
	for _, o := range d.Openings {
		if err := accounts.Open(o.Account); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

func (pr *PASTBuilder) processTransactions(ctx context.Context, accounts accounts, amounts amounts.Amounts, d *ast.Day) []error {
	var errors []error
	for _, t := range d.Transactions {
		for _, p := range t.Postings {
			if !accounts.IsOpen(p.Credit) {
				errors = append(errors, Error{t, fmt.Sprintf("credit account %s is not open", p.Credit)})
			}
			if !accounts.IsOpen(p.Debit) {
				errors = append(errors, Error{t, fmt.Sprintf("debit account %s is not open", p.Debit)})
			}
			amounts.Book(p.Credit, p.Debit, p.Amount, p.Commodity)
		}
	}
	return errors
}

func (pr *PASTBuilder) processValues(ctx context.Context, accounts accounts, amounts amounts.Amounts, d *ast.Day) ([]*ast.Transaction, []error) {
	var errors []error
	var transactions []*ast.Transaction
	for _, v := range d.Values {
		if !accounts.IsOpen(v.Account) {
			errors = append(errors, Error{v, "account is not open"})
		}
		valAcc := pr.Context.ValuationAccountFor(v.Account)
		posting := ast.NewPostingWithTargets(valAcc, v.Account, v.Commodity, v.Amount.Sub(amounts.Amount(v.Account, v.Commodity)), []*journal.Commodity{v.Commodity})
		amounts.Book(posting.Credit, posting.Debit, posting.Amount, posting.Commodity)
		transactions = append(transactions, &ast.Transaction{
			Date:        v.Date,
			Description: fmt.Sprintf("Valuation adjustment for %v in %v", v.Commodity, v.Account),
			Tags:        nil,
			Postings:    []ast.Posting{posting},
		})
	}
	return transactions, errors
}

func (pr *PASTBuilder) processAssertions(ctx context.Context, accounts accounts, amts amounts.Amounts, d *ast.Day) []error {
	var errors []error
	for _, a := range d.Assertions {
		if !accounts.IsOpen(a.Account) {
			errors = append(errors, Error{a, "account is not open"})
		}
		position := amounts.CommodityAccount{Account: a.Account, Commodity: a.Commodity}
		if va, ok := amts[position]; !ok || !va.Equal(a.Amount) {
			errors = append(errors, Error{a, fmt.Sprintf("assertion failed: account %s has %s %s", a.Account, va, position.Commodity)})
		}
	}
	return errors
}

func (pr *PASTBuilder) processClosings(ctx context.Context, accounts accounts, amounts amounts.Amounts, d *ast.Day) []error {
	var errors []error
	for _, c := range d.Closings {
		for pos, amount := range amounts {
			if pos.Account != c.Account {
				continue
			}
			if !amount.IsZero() {
				errors = append(errors, Error{c, "account has nonzero position"})
			}
			delete(amounts, pos)
		}
		if err := accounts.Close(c.Account); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

// accounts keeps track of open accounts.
type accounts map[*journal.Account]bool

// Open opens an account.
func (oa accounts) Open(a *journal.Account) error {
	if oa[a] {
		return fmt.Errorf("account %v is already open", a)
	}
	oa[a] = true
	return nil
}

// Close closes an account.
func (oa accounts) Close(a *journal.Account) error {
	if !oa[a] {
		return fmt.Errorf("account %v is already closed", a)
	}
	delete(oa, a)
	return nil
}

// IsOpen returns whether an account is open.
func (oa accounts) IsOpen(a *journal.Account) bool {
	return oa[a] || a.Type() == journal.EQUITY
}
