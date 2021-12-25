package process

import (
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
func (pr *ASTBuilder) ASTFromPath(p string) (*ast.AST, error) {
	par := parser.RecursiveParser{
		File:    p,
		Context: pr.Context,
	}
	results := par.Parse()
	var b = &ast.AST{
		Context: pr.Context,
		Days:    make(map[time.Time]*ast.Day),
	}
	for res := range results {
		switch t := res.(type) {
		case error:
			return nil, t
		case *ast.Open:
			b.AddOpen(t)
		case *ast.Price:
			b.AddPrice(t)
		case *ast.Transaction:
			b.AddTransaction(t)
		case *ast.Assertion:
			b.AddAssertion(t)
		case *ast.Value:
			b.AddValue(t)
		case *ast.Close:
			b.AddClose(t)
		default:
			return nil, fmt.Errorf("unknown: %#v", t)
		}
	}
	return b, nil
}
