package process

import (
	"context"
	"fmt"
	"time"

	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
	"github.com/sboehler/knut/lib/journal/ast/parser"
)

// ASTBuilder builds an abstract syntax tree.
type ASTBuilder struct {
	Context journal.Context

	Expand bool
	Filter journal.Filter
}

// ASTFromPath reads directives from the given channel and
// builds a Ledger if successful.
func (pr *ASTBuilder) ASTFromPath(ctx context.Context, p string) (*ast.AST, error) {
	par := parser.RecursiveParser{
		File:    p,
		Context: pr.Context,
	}
	res := &ast.AST{
		Context: pr.Context,
		Days:    make(map[time.Time]*ast.Day),
	}
	resCh, errCh := par.Parse(ctx)

	for resCh != nil || errCh != nil {
		select {
		case err, ok := <-errCh:
			if !ok {
				errCh = nil
				break
			}
			return nil, err

		case d, ok := <-resCh:
			if !ok {
				resCh = nil
				break
			}
			switch t := d.(type) {
			case error:
				return nil, t
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
				return nil, fmt.Errorf("unknown: %#v", t)
			}
		}
	}
	return res, nil
}

// ASTExpander expands and filters the given AST.
type ASTExpander struct {
	Expand bool
	Filter journal.Filter
}

// Process processes the AST and returns a new copy.
func (pr *ASTExpander) Process(a *ast.AST) *ast.AST {
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
			t := t.Clone()
			t.Postings = filtered
		}
		a.AddTransaction(t)
	}
}
