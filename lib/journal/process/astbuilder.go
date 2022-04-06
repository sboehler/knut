package process

import (
	"context"
	"fmt"
	"time"

	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
	"github.com/sboehler/knut/lib/journal/ast/parser"
)

// JournalSource emits journal data in daily batches.
type JournalSource struct {
	Context journal.Context

	Path   string
	Expand bool
	Filter journal.Filter
}

// Source is a source of days.
func (pr *JournalSource) Source(ctx context.Context, outCh chan<- *ast.Day) error {
	a := &ast.AST{
		Context: pr.Context,
		Days:    make(map[time.Time]*ast.Day),
	}
	p := parser.RecursiveParser{
		Context: pr.Context,
		File:    pr.Path,
	}

	ch, errCh := p.Parse(ctx)
	for {
		d, ok, err := cpr.Get(ch, errCh)
		if err != nil {
			return err
		}
		if !ok {
			break
		}
		switch t := d.(type) {

		case *ast.Open:
			a.AddOpen(t)

		case *ast.Price:
			a.AddPrice(t)

		case *ast.Transaction:
			filtered := t.FilterPostings(pr.Filter)
			if len(filtered) == 0 {
				break
			}
			if len(filtered) < len(t.Postings()) {
				tb := t.ToBuilder()
				tb.Postings = filtered
				t = tb.Build()
			}
			if t.Accrual() != nil {
				for _, ts := range t.Accrual().Expand(t) {
					a.AddTransaction(ts)
				}
			} else {
				a.AddTransaction(t)
			}

		case *ast.Assertion:
			if !pr.Filter.MatchAccount(t.Account) {
				break
			}
			if !pr.Filter.MatchCommodity(t.Commodity) {
				break
			}
			a.AddAssertion(t)

		case *ast.Value:
			if !pr.Filter.MatchAccount(t.Account) {
				break
			}
			if !pr.Filter.MatchCommodity(t.Commodity) {
				break
			}
			a.AddValue(t)

		case *ast.Close:
			if !pr.Filter.MatchAccount(t.Account) {
				break
			}
			a.AddClose(t)

		default:
			return fmt.Errorf("unknown: %#v", t)
		}
	}
	for _, d := range a.SortedDays() {
		if err := cpr.Push(ctx, outCh, d); err != nil {
			return err
		}
	}
	return nil
}
