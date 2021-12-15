package process

import (
	"sort"
	"time"

	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
	"github.com/sboehler/knut/lib/journal/past"
)

// Processor processes ASTs.
type Processor struct {
	Filter journal.Filter
	Expand bool
}

// Process processes an AST to a PAST
func (pr Processor) Process(a *ast.AST) *past.PAST {
	var astCp = &ast.AST{
		Days:    make(map[time.Time]*ast.Day),
		Context: a.Context,
	}
	for d, day := range a.Days {
		dayCp := astCp.Day(d)

		dayCp.Openings = make([]*ast.Open, len(day.Openings))
		copy(dayCp.Openings, day.Openings)

		dayCp.Prices = make([]*ast.Price, len(day.Prices))
		copy(dayCp.Prices, day.Prices)

		for _, trx := range day.Transactions {
			pr.ProcessTransaction(astCp, trx)
		}

		for _, a := range day.Assertions {
			pr.ProcessAssertion(astCp, a)
		}

		dayCp.Closings = make([]*ast.Close, len(day.Closings))
		copy(dayCp.Closings, day.Closings)
	}
	var sorted []*ast.Day
	for _, day := range astCp.Days {
		sorted = append(sorted, day)
	}

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Less(sorted[j])
	})

	// TODO: process values

	return &past.PAST{
		Context: a.Context,
		Days:    sorted,
	}

}

// ProcessTransaction adds a transaction directive.
func (pr *Processor) ProcessTransaction(a *ast.AST, t *ast.Transaction) {
	if pr.Expand && len(t.AddOns) > 0 {
		for _, addOn := range t.AddOns {
			switch acc := addOn.(type) {
			case *ast.Accrual:
				for _, ts := range acc.Expand(t) {
					pr.ProcessTransaction(a, ts)
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
			a.AddTransaction(&ast.Transaction{
				Range:       t.Range,
				Date:        t.Date,
				Description: t.Description,
				Postings:    filtered,
				Tags:        t.Tags,
			})
		}
	}
}

// ProcessAssertion adds an assertion directive.
func (pr *Processor) ProcessAssertion(as *ast.AST, a *ast.Assertion) {
	if pr.Filter.MatchAccount(a.Account) && pr.Filter.MatchCommodity(a.Commodity) {
		as.AddAssertion(a)
	}
}
