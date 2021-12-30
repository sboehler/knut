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
