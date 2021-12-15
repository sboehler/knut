package past

import (
	"sort"
	"time"

	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
)

// Processor processes ASTs.
type Processor struct {
	Filter journal.Filter
}

// Process processes an AST to a PAST
func (Processor) Process(a *ast.AST1) *ast.PAST {
	var astp = &ast.AST1{
		Days:    make(map[time.Time]*ast.Day),
		Context: a.Context,
	}
	for d, day := range a.Days {
		dayp := astp.Day(d)

		// TODO: filter directives
		dayp.Openings = day.Openings
		dayp.Closings = day.Closings
		dayp.Assertions = day.Assertions
		dayp.Prices = day.Prices

		for _, trx := range day.Transactions {
			// TODO: process trx
			dayp.Transactions = append(dayp.Transactions, trx)
		}
	}
	var sorted []*ast.Day
	for _, day := range astp.Days {
		sorted = append(sorted, day)
	}

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Less(sorted[j])
	})

	// TODO: process values

	return &ast.PAST{
		Context: a.Context,
		Days:    sorted,
	}

}
