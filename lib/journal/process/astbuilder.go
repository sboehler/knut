package process

import (
	"context"
	"fmt"
	"time"

	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
)

// ASTBuilder builds an abstract syntax tree.
type ASTBuilder struct {
	Context journal.Context

	resCh chan *ast.AST
	errCh chan error
}

// BuildAST reads directives from the given channel and
// builds a Ledger if successful.
func (pr *ASTBuilder) BuildAST(ctx context.Context, inCh <-chan ast.Directive) (<-chan *ast.AST, <-chan error) {
	pr.resCh = make(chan *ast.AST)
	pr.errCh = make(chan error)
	go func() {
		defer close(pr.resCh)
		defer close(pr.errCh)
		res := &ast.AST{
			Context: pr.Context,
			Days:    make(map[time.Time]*ast.Day),
		}
		for inCh != nil {
			select {

			case d, ok := <-inCh:
				if !ok {
					inCh = nil
					break
				}
				switch t := d.(type) {
				case *ast.Open:
					res.AddOpen(t)
				case *ast.Price:
					res.AddPrice(t)
				case *ast.Transaction:
					res.AddTransaction(t)
				case *ast.Assertion:
					res.AddAssertion(t)
				case *ast.Value:
					res.AddValue(t)
				case *ast.Close:
					res.AddClose(t)
				default:
					select {
					case pr.errCh <- fmt.Errorf("unknown: %#v", t):
					case <-ctx.Done():
						return
					}
				}

			case <-ctx.Done():
				return
			}
		}
		select {
		case pr.resCh <- res:
		case <-ctx.Done():
		}
	}()
	return pr.resCh, pr.errCh
}

// ASTExpander expands and filters the given AST.
type ASTExpander struct {
	Expand bool
	Filter journal.Filter

	resCh chan *ast.AST
	errCh chan error
}

// ExpandAndFilterAST processes the given AST.
func (pr *ASTExpander) ExpandAndFilterAST(ctx context.Context, inCh <-chan *ast.AST) (<-chan *ast.AST, <-chan error) {
	pr.resCh = make(chan *ast.AST)
	pr.errCh = make(chan error)
	go func() {
		defer close(pr.resCh)
		defer close(pr.errCh)

		for d := range inCh {
			r := pr.process(d)
			select {
			case <-ctx.Done():
				return
			case pr.resCh <- r:
			}
		}
	}()
	return pr.resCh, pr.errCh
}

// process processes the AST and returns a new copy.
func (pr *ASTExpander) process(a *ast.AST) *ast.AST {
	res := &ast.AST{
		Context: a.Context,
		Days:    make(map[time.Time]*ast.Day),
	}
	for d, astDay := range a.Days {
		day := res.Day(d)

		day.Openings = astDay.Openings
		day.Prices = astDay.Prices
		day.Closings = astDay.Closings

		for _, val := range astDay.Values {
			if pr.Filter.MatchAccount(val.Account) && pr.Filter.MatchCommodity(val.Commodity) {
				day.Values = append(day.Values, val)
			}
		}

		for _, trx := range astDay.Transactions {
			pr.expandTransaction(res, trx)
		}

		for _, a := range astDay.Assertions {
			if pr.Filter.MatchAccount(a.Account) && pr.Filter.MatchCommodity(a.Commodity) {
				day.Assertions = append(day.Assertions, a)
			}
		}
	}
	return res
}

// ProcessTransaction adds a transaction directive.
func (pr *ASTExpander) expandTransaction(a *ast.AST, t *ast.Transaction) {
	if pr.Expand && len(t.AddOns) > 0 {
		for _, addOn := range t.AddOns {
			switch acc := addOn.(type) {
			case *ast.Accrual:
				for _, ts := range acc.Expand(t) {
					pr.expandTransaction(a, ts)
				}
			default:
				panic(fmt.Sprintf("unknown addon: %#v", acc))
			}
		}
		return
	}
	var filtered []ast.Posting
	for _, p := range t.Postings {
		if p.Matches(pr.Filter) {
			filtered = append(filtered, p)
		}
	}
	if len(filtered) > 0 {
		if len(filtered) < len(t.Postings) {
			t = t.Clone()
			t.Postings = filtered
		}
		a.AddTransaction(t)
	}
}
