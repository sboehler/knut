package process

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/sboehler/knut/lib/common/amounts"
	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
	"github.com/sboehler/knut/lib/journal/past"
	"github.com/shopspring/decimal"
)

func TestPASTBuilderHappyCase(t *testing.T) {
	var (
		jctx       = journal.NewContext()
		td         = newTestData(jctx)
		astBuilder = PASTBuilder{Context: jctx}
		inCh       = make(chan *ast.AST)
		ctx        = context.Background()
		input      = &ast.AST{
			Context: jctx,
			Days: map[time.Time]*ast.Day{
				td.date1: {
					Date:         td.date1,
					Openings:     []*ast.Open{td.open1, td.open2},
					Prices:       []*ast.Price{td.price1},
					Transactions: []*ast.Transaction{td.trx1},
				},
				td.date2: {
					Date:         td.date2,
					Transactions: []*ast.Transaction{td.trx2},
				},
			},
		}
		want = []*past.Day{
			{
				AST:          input.Days[td.date1],
				Date:         td.date1,
				Transactions: []*ast.Transaction{td.trx1},
				Amounts: amounts.Amounts{
					amounts.CommodityAccount{Account: td.account1, Commodity: td.commodity1}: decimal.NewFromInt(-10),
					amounts.CommodityAccount{Account: td.account2, Commodity: td.commodity1}: decimal.NewFromInt(10),
				},
			},
			{
				AST:          input.Days[td.date2],
				Date:         td.date2,
				Transactions: []*ast.Transaction{td.trx2},
				Amounts: amounts.Amounts{
					amounts.CommodityAccount{Account: td.account1, Commodity: td.commodity1}: decimal.NewFromInt(-10),
					amounts.CommodityAccount{Account: td.account2, Commodity: td.commodity1}: decimal.NewFromInt(10),
					amounts.CommodityAccount{Account: td.account1, Commodity: td.commodity2}: decimal.NewFromInt(11),
					amounts.CommodityAccount{Account: td.account2, Commodity: td.commodity2}: decimal.NewFromInt(-11),
				},
			},
		}
	)
	resCh, errCh := astBuilder.ProcessAST(ctx, inCh)
	go func() {
		defer close(inCh)
		inCh <- input
	}()

	var got []*past.Day
	for {
		d, ok, err := cpr.Get(resCh, errCh)
		if !ok {
			break
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got = append(got, d)
	}

	if diff := cmp.Diff(got, want, cmpopts.IgnoreUnexported(journal.Context{}, journal.Commodity{}, journal.Account{})); diff != "" {
		t.Fatalf("unexpected diff (+got/-want):\n%s", diff)
	}
	if _, ok := <-resCh; ok {
		t.Fatalf("resCh is open, want closed")
	}
	if _, ok := <-errCh; ok {
		t.Fatalf("errCh is open, want closed")
	}
}

func TestPASTBuilderCanceled(t *testing.T) {
	var (
		jctx        = journal.NewContext()
		ctx, cancel = context.WithCancel(context.Background())
		astBuilder  = PASTBuilder{Context: jctx}

		inCh         chan *ast.AST
		resCh, errCh = astBuilder.ProcessAST(ctx, inCh)
	)

	cancel()

	if _, ok := <-resCh; ok {
		t.Fatalf("resCh is open, want closed")
	}
	if _, ok := <-errCh; ok {
		t.Fatalf("errCh is open, want closed")
	}
}
