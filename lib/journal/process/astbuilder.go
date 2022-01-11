package process

import (
	"context"
	"fmt"
	"time"

	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
)

// ASTBuilder builds an abstract syntax tree.
type ASTBuilder struct {
	Context journal.Context
}

// BuildAST reads directives from the given channel and
// builds a Ledger if successful.
func (pr *ASTBuilder) BuildAST(ctx context.Context, inCh <-chan ast.Directive) (<-chan *ast.AST, <-chan error) {
	resCh := make(chan *ast.AST)
	errCh := make(chan error)
	go func() {
		defer close(resCh)
		defer close(errCh)
		res := &ast.AST{
			Context: pr.Context,
			Days:    make(map[time.Time]*ast.Day),
		}
		for {
			d, ok, err := cpr.Pop(ctx, inCh)
			if err != nil {
				return
			}
			if !ok {
				cpr.Push(ctx, resCh, res)
				return
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
				if cpr.Push(ctx, errCh, fmt.Errorf("unknown: %#v", t)) != nil {
					return
				}
			}
		}
	}()
	return resCh, errCh
}

// ASTExpander expands and filters the given AST.
type ASTExpander struct {
	Expand bool
	Filter journal.Filter
}

// ExpandAndFilterAST processes the given AST.
func (pr *ASTExpander) ExpandAndFilterAST(ctx context.Context, inCh <-chan *ast.AST) (<-chan *ast.AST, <-chan error) {
	resCh := make(chan *ast.AST)
	errCh := make(chan error)
	go func() {
		defer close(resCh)
		defer close(errCh)

		for {
			d, ok, err := cpr.Pop(ctx, inCh)
			if !ok || err != nil {
				return
			}
			r := pr.process(d)
			if cpr.Push(ctx, resCh, r) != nil {
				return
			}
		}
	}()
	return resCh, errCh
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

		for _, c := range astDay.Closings {
			if pr.Filter.MatchAccount(c.Account) {
				day.Closings = append(day.Closings, c)
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
