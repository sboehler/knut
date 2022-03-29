package process

import (
	"context"
	"fmt"
	"time"

	"github.com/sboehler/knut/lib/common/amounts"
	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
	"github.com/sboehler/knut/lib/journal/past"
	"golang.org/x/sync/errgroup"
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
				)
				if err = pr.processOpenings(ctx, accounts, d); err != nil && cpr.Push(ctx, errCh, err) != nil {
					return
				}
				if err = pr.processTransactions(ctx, accounts, amounts, d); err != nil && cpr.Push(ctx, errCh, err) != nil {
					return
				}
				if transactions, err = pr.processValues(ctx, accounts, amounts, d); err != nil && cpr.Push(ctx, errCh, err) != nil {
					return
				}
				if err = pr.processAssertions(ctx, accounts, amounts, d); err != nil && cpr.Push(ctx, errCh, err) != nil {
					return
				}
				if err = pr.processClosings(ctx, accounts, amounts, d); err != nil && cpr.Push(ctx, errCh, err) != nil {
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

// Process2 processes days.
func (pr *PASTBuilder) Process2(ctx context.Context, g *errgroup.Group, inCh <-chan *ast.Day) <-chan *ast.Day {

	resCh := make(chan *ast.Day, 100)

	g.Go(func() error {
		defer close(resCh)

		amounts := make(amounts.Amounts)
		accounts := make(accounts)

		for {
			d, ok, err := cpr.Pop(ctx, inCh)
			if err != nil {
				return err
			}
			if !ok {
				break
			}
			var transactions []*ast.Transaction
			if err := pr.processOpenings(ctx, accounts, d); err != nil {
				return err
			}
			if err := pr.processTransactions(ctx, accounts, amounts, d); err != nil {
				return err
			}
			if transactions, err = pr.processValues(ctx, accounts, amounts, d); err != nil {
				return err
			}
			if err = pr.processAssertions(ctx, accounts, amounts, d); err != nil {
				return err
			}
			if err = pr.processClosings(ctx, accounts, amounts, d); err != nil {
				return err
			}

			d.Transactions = append(d.Transactions, transactions...)
			d.Amounts = amounts.Clone()

			if err := cpr.Push(ctx, resCh, d); err != nil {
				return err
			}
		}
		return nil
	})
	return resCh
}

func (pr *PASTBuilder) processOpenings(ctx context.Context, accounts accounts, d *ast.Day) error {
	for _, o := range d.Openings {
		if err := accounts.Open(o.Account); err != nil {
			return err
		}
	}
	return nil
}

func (pr *PASTBuilder) processTransactions(ctx context.Context, accounts accounts, amounts amounts.Amounts, d *ast.Day) error {
	for _, t := range d.Transactions {
		for _, p := range t.Postings {
			if !accounts.IsOpen(p.Credit) {
				return Error{t, fmt.Sprintf("credit account %s is not open", p.Credit)}
			}
			if !accounts.IsOpen(p.Debit) {
				return Error{t, fmt.Sprintf("debit account %s is not open", p.Debit)}
			}
			amounts.Book(p.Credit, p.Debit, p.Amount, p.Commodity)
		}
	}
	return nil
}

func (pr *PASTBuilder) processValues(ctx context.Context, accounts accounts, amounts amounts.Amounts, d *ast.Day) ([]*ast.Transaction, error) {
	var transactions []*ast.Transaction
	for _, v := range d.Values {
		if !accounts.IsOpen(v.Account) {
			return nil, Error{v, "account is not open"}
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
	return transactions, nil
}

func (pr *PASTBuilder) processAssertions(ctx context.Context, accounts accounts, amts amounts.Amounts, d *ast.Day) error {
	for _, a := range d.Assertions {
		if !accounts.IsOpen(a.Account) {
			return Error{a, "account is not open"}
		}
		position := amounts.CommodityAccount{Account: a.Account, Commodity: a.Commodity}
		if va, ok := amts[position]; !ok || !va.Equal(a.Amount) {
			return Error{a, fmt.Sprintf("assertion failed: account %s has %s %s", a.Account, va, position.Commodity)}
		}
	}
	return nil
}

func (pr *PASTBuilder) processClosings(ctx context.Context, accounts accounts, amounts amounts.Amounts, d *ast.Day) error {
	for _, c := range d.Closings {
		for pos, amount := range amounts {
			if pos.Account != c.Account {
				continue
			}
			if !amount.IsZero() {
				return Error{c, "account has nonzero position"}
			}
			delete(amounts, pos)
		}
		if err := accounts.Close(c.Account); err != nil {
			return err
		}
	}
	return nil
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

// Booker processes ASTs.
type Booker struct {

	// The context of this journal.
	Context journal.Context

	amounts  amounts.Amounts
	accounts accounts
	date     time.Time
	send     bool
}

var _ ast.Processor = (*Booker)(nil)

// Process checks assertions and the usage of open and closed accounts.
// It also resolves Value directives and converts them to transactions.
func (pr *Booker) Process(ctx context.Context, d ast.Dated, next func(ast.Dated) bool) error {

	if pr.amounts == nil {
		pr.amounts = make(amounts.Amounts)
		pr.accounts = make(accounts)
	}

	if pr.date != d.Date && pr.send {
		if !next(ast.Dated{Date: pr.date, Elem: pr.amounts.Clone()}) {
			return nil
		}
		pr.date = d.Date
		pr.send = false
	}

	switch t := d.Elem.(type) {

	case *ast.Open:
		if err := pr.accounts.Open(t.Account); err != nil {
			return err
		}

	case *ast.Transaction:
		pr.send = true
		for _, p := range t.Postings {
			if !pr.accounts.IsOpen(p.Credit) {
				return Error{t, fmt.Sprintf("credit account %s is not open", p.Credit)}
			}
			if !pr.accounts.IsOpen(p.Debit) {
				return Error{t, fmt.Sprintf("debit account %s is not open", p.Debit)}
			}
			pr.amounts.Book(p.Credit, p.Debit, p.Amount, p.Commodity)
		}
		next(d)

	case *ast.Value:
		pr.send = true
		if !pr.accounts.IsOpen(t.Account) {
			return Error{t, "account is not open"}
		}
		valAcc := pr.Context.ValuationAccountFor(t.Account)
		posting := ast.NewPostingWithTargets(valAcc, t.Account, t.Commodity, t.Amount.Sub(pr.amounts.Amount(t.Account, t.Commodity)), []*journal.Commodity{t.Commodity})
		pr.amounts.Book(posting.Credit, posting.Debit, posting.Amount, posting.Commodity)
		next(ast.Dated{
			Date: t.Date,
			Elem: &ast.Transaction{
				Date:        t.Date,
				Description: fmt.Sprintf("Valuation adjustment for %v in %v", t.Commodity, t.Account),
				Postings:    []ast.Posting{posting},
			}})

	case *ast.Assertion:
		if !pr.accounts.IsOpen(t.Account) {
			return Error{t, "account is not open"}
		}
		position := amounts.CommodityAccount{Account: t.Account, Commodity: t.Commodity}
		if va, ok := pr.amounts[position]; !ok || !va.Equal(t.Amount) {
			return Error{t, fmt.Sprintf("assertion failed: account %s has %s %s", t.Account, va, position.Commodity)}
		}

	case *ast.Close:
		for pos, amount := range pr.amounts {
			if pos.Account != t.Account {
				continue
			}
			if !amount.IsZero() {
				return Error{t, "account has nonzero position"}
			}
			delete(pr.amounts, pos)
		}
		if err := pr.accounts.Close(t.Account); err != nil {
			return err
		}

	default:
		next(d)
	}

	return nil
}

// Finalize implements Finalize.
func (pr *Booker) Finalize(ctx context.Context, next func(ast.Dated) bool) error {
	if pr.send {
		next(ast.Dated{Date: pr.date, Elem: pr.amounts.Clone()})
	}
	return nil
}
