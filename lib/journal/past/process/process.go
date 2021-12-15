package process

import (
	"sort"
	"time"

	"github.com/sboehler/knut/lib/balance"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
	"github.com/sboehler/knut/lib/journal/past"
)

// Processor processes ASTs.
type Processor struct {

	// Filter applies the given filter to postings of transactions
	// and assertions.
	Filter journal.Filter

	// Expand controls whether Accrual add-ons are expanded.
	Expand bool
}

// Process processes an AST to a PAST. It check assertions
// and the usage of open and closed accounts. It will also
// resolve Value directives and convert them to transactions.
func (pr Processor) Process(a *ast.AST) (*past.PAST, error) {
	var astCp = &ast.AST{
		Context: a.Context,
		Days:    make(map[time.Time]*ast.Day),
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

	var (
		pAST = &past.PAST{
			Context: a.Context,
			Days:    sorted,
		}
		bal   = balance.New(a.Context, nil)
		steps = []past.Processor{
			balance.AccountOpener{Balance: bal},
			balance.TransactionBooker{Balance: bal},
			balance.ValueBooker{Balance: bal},
			balance.Asserter{Balance: bal},
			balance.AccountCloser{Balance: bal},
		}
	)

	if err := past.Sync(pAST, steps); err != nil {
		return nil, err
	}
	return pAST, nil

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
