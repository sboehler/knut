package process

import (
	"context"
	"fmt"

	"github.com/sboehler/knut/lib/common/amounts"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
	"github.com/sboehler/knut/lib/journal/past"
)

// PASTBuilder processes ASTs.
type PASTBuilder struct {

	// The context of this journal.
	Context journal.Context

	// Filter applies the given filter to postings of transactions
	// and assertions.
	Filter journal.Filter

	// Expand controls whether Accrual add-ons are expanded.
	Expand bool

	// Private fields to hold and share channels between methods.
	errCh chan error
	resCh chan *past.Day

	amounts      amounts.Amounts
	accounts     accounts
	transactions []*ast.Transaction
}

// FromAST processes an AST to a PAST. It check assertions
// and the usage of open and closed accounts. It will also
// resolve Value directives and convert them to transactions.
func (pr PASTBuilder) FromAST(ctx context.Context, a *ast.AST) (*past.PAST, error) {
	dayCh, errCh := pr.StreamFromAST(ctx, a)
	var days []*past.Day
	for dayCh != nil || errCh != nil {
		select {
		case day, ok := <-dayCh:
			if !ok {
				dayCh = nil
				break
			}
			days = append(days, day)

		case err, ok := <-errCh:
			if !ok {
				errCh = nil
				break
			}
			return nil, err
		}
	}
	return &past.PAST{
		Context: a.Context,
		Days:    days,
	}, nil
}

// errOrExit will pass the error to errCh if it is not nil. It will return
// false if the error was successfully passed or if the error is nil. It
// returns true if the error couldn't be passed because the context was canceled.
func (pr *PASTBuilder) errOrExit(ctx context.Context, err error) bool {
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

// StreamFromAST processes an AST to a stream of past.Day. It check assertions
// and the usage of open and closed accounts. It will also
// resolve Value directives and convert them to transactions.
func (pr *PASTBuilder) StreamFromAST(ctx context.Context, a *ast.AST) (<-chan *past.Day, <-chan error) {

	pr.errCh = make(chan error)
	pr.resCh = make(chan *past.Day, 100)

	go func() {
		defer close(pr.resCh)
		defer close(pr.errCh)

		pr.amounts = make(amounts.Amounts)
		pr.accounts = make(accounts)

		for _, d := range a.SortedDays() {
			if pr.processOpenings(ctx, d) {
				return
			}
			if pr.processTransactions(ctx, d) {
				return
			}
			if pr.processValues(ctx, d) {
				return
			}
			if pr.processAssertions(ctx, d) {
				return
			}
			if pr.processClosings(ctx, d) {
				return
			}
			res := &past.Day{
				Date:         d.Date,
				AST:          d,
				Transactions: pr.transactions,
				Amounts:      pr.amounts,
			}
			pr.amounts = pr.amounts.Clone()
			pr.transactions = nil
			select {
			case pr.resCh <- res:
			case <-ctx.Done():
				return
			}
		}
	}()
	return pr.resCh, pr.errCh
}

// ProcessAST processes an AST to a stream of past.Day. It check assertions
// and the usage of open and closed accounts. It will also
// resolve Value directives and convert them to transactions.
func (pr *PASTBuilder) ProcessAST(ctx context.Context, inCh <-chan *ast.AST) (<-chan *past.Day, <-chan error) {

	pr.errCh = make(chan error)
	pr.resCh = make(chan *past.Day, 100)

	go func() {
		defer close(pr.resCh)
		defer close(pr.errCh)

		for a := range inCh {

			pr.amounts = make(amounts.Amounts)
			pr.accounts = make(accounts)

			for _, d := range a.SortedDays() {
				if pr.processOpenings(ctx, d) {
					return
				}
				if pr.processTransactions(ctx, d) {
					return
				}
				if pr.processValues(ctx, d) {
					return
				}
				if pr.processAssertions(ctx, d) {
					return
				}
				if pr.processClosings(ctx, d) {
					return
				}
				res := &past.Day{
					Date:         d.Date,
					AST:          d,
					Transactions: pr.transactions,
					Amounts:      pr.amounts,
				}
				pr.amounts = pr.amounts.Clone()
				pr.transactions = nil
				select {
				case pr.resCh <- res:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return pr.resCh, pr.errCh
}

func (pr *PASTBuilder) processOpenings(ctx context.Context, d *ast.Day) bool {
	for _, o := range d.Openings {
		if err := pr.accounts.Open(o.Account); pr.errOrExit(ctx, err) {
			return true
		}
	}
	return false
}

func (pr *PASTBuilder) processTransactions(ctx context.Context, d *ast.Day) bool {
	for _, t := range d.Transactions {
		for _, p := range t.Postings {
			if !pr.accounts.IsOpen(p.Credit) {
				if pr.errOrExit(ctx, Error{t, fmt.Sprintf("credit account %s is not open", p.Credit)}) {
					return true
				}
			}
			if !pr.accounts.IsOpen(p.Debit) {
				if pr.errOrExit(ctx, Error{t, fmt.Sprintf("debit account %s is not open", p.Debit)}) {
					return true
				}
			}
			pr.amounts.Book(p.Credit, p.Debit, p.Amount, p.Commodity)
		}
		pr.transactions = append(pr.transactions, t)
	}
	return false
}

func (pr *PASTBuilder) processValues(ctx context.Context, d *ast.Day) bool {
	for _, v := range d.Values {
		if !pr.accounts.IsOpen(v.Account) {
			if pr.errOrExit(ctx, Error{v, "account is not open"}) {
				return true
			}
		}
		valAcc, err := pr.Context.ValuationAccountFor(v.Account)
		if pr.errOrExit(ctx, err) {
			return true
		}
		posting := ast.NewPosting(valAcc, v.Account, v.Commodity, v.Amount.Sub(pr.amounts.Amount(v.Account, v.Commodity)))
		pr.amounts.Book(posting.Credit, posting.Debit, posting.Amount, posting.Commodity)
		pr.transactions = append(pr.transactions, &ast.Transaction{
			Date:        v.Date,
			Description: fmt.Sprintf("Valuation adjustment for %v in %v", v.Commodity, v.Account),
			Tags:        nil,
			Postings:    []ast.Posting{posting},
		})
	}
	return false
}

func (pr *PASTBuilder) processAssertions(ctx context.Context, d *ast.Day) bool {
	// check assertions
	for _, a := range d.Assertions {
		if !pr.accounts.IsOpen(a.Account) {
			if pr.errOrExit(ctx, Error{a, "account is not open"}) {
				return true
			}
		}
		position := amounts.CommodityAccount{Account: a.Account, Commodity: a.Commodity}
		if va, ok := pr.amounts[position]; !ok || !va.Equal(a.Amount) {
			if pr.errOrExit(ctx, Error{a, fmt.Sprintf("assertion failed: account %s has %s %s", a.Account, va, position.Commodity)}) {
				return true
			}
		}
	}
	return false
}

func (pr *PASTBuilder) processClosings(ctx context.Context, d *ast.Day) bool {
	// close accounts
	for _, c := range d.Closings {
		for pos, amount := range pr.amounts {
			if pos.Account != c.Account {
				continue
			}
			if !amount.IsZero() && pr.errOrExit(ctx, Error{c, "account has nonzero position"}) {
				return true
			}
			delete(pr.amounts, pos)
		}
		if err := pr.accounts.Close(c.Account); pr.errOrExit(ctx, err) {
			return true
		}
	}
	return false
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
	if oa[a] {
		return true
	}
	return a.Type() == journal.EQUITY
}
