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

	Path     string
	Expand   bool
	Filter   journal.Filter
	AutoLoad bool

	ast *ast.AST
}

func (js *JournalSource) Load(ctx context.Context) error {
	js.ast = ast.New(js.Context)
	p := parser.RecursiveParser{
		Context: js.Context,
		File:    js.Path,
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
			js.ast.AddOpen(t)

		case *ast.Price:
			js.ast.AddPrice(t)

		case *ast.Transaction:
			filtered := t.FilterPostings(js.Filter)
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
					js.ast.AddTransaction(ts)
				}
			} else {
				js.ast.AddTransaction(t)
			}

		case *ast.Assertion:
			if !js.Filter.MatchAccount(t.Account) {
				break
			}
			if !js.Filter.MatchCommodity(t.Commodity) {
				break
			}
			js.ast.AddAssertion(t)

		case *ast.Value:
			if !js.Filter.MatchAccount(t.Account) {
				break
			}
			if !js.Filter.MatchCommodity(t.Commodity) {
				break
			}
			js.ast.AddValue(t)

		case *ast.Close:
			if !js.Filter.MatchAccount(t.Account) {
				break
			}
			js.ast.AddClose(t)

		default:
			return fmt.Errorf("unknown: %#v", t)
		}
	}
	return nil
}

func (js JournalSource) Min() time.Time {
	return js.ast.Min()
}

func (js JournalSource) Max() time.Time {
	return js.ast.Max()
}

func (js JournalSource) Source(ctx context.Context, outCh chan<- *ast.Day) error {
	if js.AutoLoad {
		if err := js.Load(ctx); err != nil {
			return err
		}
	}
	for _, d := range js.ast.SortedDays() {
		if err := cpr.Push(ctx, outCh, d); err != nil {
			return err
		}
	}
	return nil
}
