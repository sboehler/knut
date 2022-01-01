package process

import (
	"context"
	"fmt"
	"time"

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

// StreamFromAST processes an AST to a stream of past.Day. It check assertions
// and the usage of open and closed accounts. It will also
// resolve Value directives and convert them to transactions.
func (pr *PASTBuilder) StreamFromAST(ctx context.Context, a *ast.AST) (<-chan *past.Day, <-chan error) {
	expandedAST := pr.expandAST(a)

	var (
		errCh = make(chan error)
		resCh = make(chan *past.Day, 100)

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
		var (
			sortedAST = expandedAST.SortedDays()

			amounts  = make(past.Amounts)
			accounts = make(past.Accounts)
		)
		for _, d := range sortedAST {
			var transactions []*ast.Transaction

			// open accounts
			for _, o := range d.Openings {
				if err := accounts.Open(o.Account); err != nil && errOrExit(err) {
					return
				}
			}

			// check and book transactions
			for _, t := range d.Transactions {
				for _, p := range t.Postings {
					if !accounts.IsOpen(p.Credit) {
						if errOrExit(Error{t, fmt.Sprintf("credit account %s is not open", p.Credit)}) {
							return
						}
					}
					if !accounts.IsOpen(p.Debit) {
						if errOrExit(Error{t, fmt.Sprintf("debit account %s is not open", p.Debit)}) {
							return
						}
					}
					amounts.Book(p.Credit, p.Debit, p.Amount, p.Commodity)
				}
				transactions = append(transactions, t)
			}

			// check and book value directives
			if dayA, ok := a.Days[d.Date]; ok {
				for _, v := range dayA.Values {
					if !accounts.IsOpen(v.Account) {
						if errOrExit(Error{v, "account is not open"}) {
							return
						}
					}
					t, err := pr.processValue(amounts, v)
					if err != nil {
						errOrExit(err)
						return
					}
					for _, p := range t.Postings {
						if !accounts.IsOpen(p.Credit) {
							if errOrExit(Error{t, fmt.Sprintf("credit account %s is not open", p.Credit)}) {
								return
							}
						}
						if !accounts.IsOpen(p.Debit) {
							if errOrExit(Error{t, fmt.Sprintf("debit account %s is not open", p.Debit)}) {
								return
							}
						}
						amounts.Book(p.Credit, p.Debit, p.Amount, p.Commodity)
					}
					transactions = append(transactions, t)
				}
			}

			// check assertions
			for _, a := range d.Assertions {
				if !accounts.IsOpen(a.Account) {
					if errOrExit(Error{a, "account is not open"}) {
						return
					}
				}
				position := past.CommodityAccount{Account: a.Account, Commodity: a.Commodity}
				if va, ok := amounts[position]; !ok || !va.Equal(a.Amount) {
					if errOrExit(Error{a, fmt.Sprintf("assertion failed: account %s has %s %s", a.Account, va, position.Commodity)}) {
						return
					}
				}
			}

			// close accounts
			for _, c := range d.Closings {
				for pos, amount := range amounts {
					if pos.Account != c.Account {
						continue
					}
					if !amount.IsZero() && errOrExit(Error{c, "account has nonzero position"}) {
						return
					}
					delete(amounts, pos)
				}
				if err := accounts.Close(c.Account); err != nil && errOrExit(err) {
					return
				}
			}
			res := &past.Day{
				Date:         d.Date,
				AST:          a.Days[d.Date], // possibly nil
				Transactions: transactions,
				Amounts:      amounts,
			}
			amounts = amounts.Clone()
			select {
			case resCh <- res:
			case <-ctx.Done():
				return
			}
		}
	}()
	return resCh, errCh
}

func (pr *PASTBuilder) expandAST(a *ast.AST) *ast.AST {
	res := &ast.AST{
		Context: pr.Context,
		Days:    make(map[time.Time]*ast.Day),
	}
	for d, astDay := range a.Days {
		day := res.Day(d)

		day.Openings = astDay.Openings
		day.Prices = astDay.Prices
		day.Closings = astDay.Closings

		for _, trx := range astDay.Transactions {
			pr.expandTransaction(res, trx)
		}

		for _, a := range astDay.Assertions {
			pr.addAssertions(res, a)
		}
	}
	return res
}

// ProcessTransaction adds a transaction directive.
func (pr *PASTBuilder) expandTransaction(a *ast.AST, t *ast.Transaction) {
	if pr.Expand && len(t.AddOns) > 0 {
		for _, addOn := range t.AddOns {
			switch acc := addOn.(type) {
			case *ast.Accrual:
				for _, ts := range acc.Expand(t) {
					pr.expandTransaction(a, ts)
				}
			}
		}
	} else {
		var filtered []ast.Posting
		for _, p := range t.Postings {
			if p.Matches(pr.Filter) {
				filtered = append(filtered, p)
			}
		}
		if len(filtered) == len(t.Postings) {
			a.AddTransaction(t)
		} else if len(filtered) > 0 && len(filtered) < len(t.Postings) {
			tn := t.Clone()
			tn.Postings = filtered
			a.AddTransaction(tn)
		}
	}
}

// ProcessAssertion adds an assertion directive.
func (pr *PASTBuilder) addAssertions(as *ast.AST, a *ast.Assertion) {
	if pr.Filter.MatchAccount(a.Account) && pr.Filter.MatchCommodity(a.Commodity) {
		as.AddAssertion(a)
	}
}

func (pr *PASTBuilder) processValue(bal past.Amounts, v *ast.Value) (*ast.Transaction, error) {
	valAcc, err := pr.Context.ValuationAccountFor(v.Account)
	if err != nil {
		return nil, err
	}
	return &ast.Transaction{
		Date:        v.Date,
		Description: fmt.Sprintf("Valuation adjustment for %v in %v", v.Commodity, v.Account),
		Tags:        nil,
		Postings: []ast.Posting{
			ast.NewPosting(valAcc, v.Account, v.Commodity, v.Amount.Sub(bal.Amount(v.Account, v.Commodity))),
		},
	}, nil
}
