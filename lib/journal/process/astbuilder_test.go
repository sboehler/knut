package process

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/sboehler/knut/lib/common/date"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
	"github.com/shopspring/decimal"
)

type DummyDirective struct{}

func (d DummyDirective) Position() ast.Range {
	return ast.Range{}
}

type TestData struct {
	date1, date2           time.Time
	account1, account2     *journal.Account
	commodity1, commodity2 *journal.Commodity
	open1, open2           *ast.Open
	price1                 *ast.Price
	trx1                   *ast.Transaction
}

func newTestData(jctx journal.Context) TestData {
	var (
		date1      = date.Date(2022, 1, 4)
		date2      = date.Date(2022, 1, 5)
		account1   = jctx.Account("Assets:Account")
		account2   = jctx.Account("Assets:Other")
		commodity1 = jctx.Commodity("COM")
		commodity2 = jctx.Commodity("TGT")
		price1     = &ast.Price{
			Date:      date1,
			Commodity: commodity1,
			Target:    commodity2,
			Price:     decimal.NewFromInt(4),
		}
		open1 = &ast.Open{Date: date2, Account: account1}
		open2 = &ast.Open{Date: date2, Account: account2}
		trx1  = &ast.Transaction{
			Date:        date2,
			Description: "foo",
			Postings: []ast.Posting{
				ast.NewPosting(account1, account2, commodity1, decimal.NewFromInt(10)),
			},
		}
	)
	return TestData{
		date1, date2,
		account1, account2,
		commodity1, commodity2,
		open1, open2,
		price1,
		trx1,
	}
}

func TestASTBuilderHappyCase(t *testing.T) {
	var (
		jctx       = journal.NewContext()
		td         = newTestData(jctx)
		astBuilder = ASTBuilder{Context: jctx}
		inCh       = make(chan ast.Directive)
		ctx        = context.Background()
		want       = &ast.AST{
			Context: jctx,
			Days: map[time.Time]*ast.Day{
				td.date1: {Date: td.date1, Prices: []*ast.Price{td.price1}},
				td.date2: {Date: td.date2, Openings: []*ast.Open{td.open1}},
			},
		}
	)
	resCh, errCh := astBuilder.BuildAST(ctx, inCh)
	go func() {
		defer close(inCh)
		inCh <- td.price1
		inCh <- td.open1
	}()
	got, ok := <-resCh
	if !ok {
		t.Fatalf("ok = false, want ok = true")
	}
	if diff := cmp.Diff(got, want, cmpopts.IgnoreUnexported(journal.Context{}, journal.Commodity{}, journal.Account{})); diff != "" {
		t.Fatalf(diff)
	}
	if _, ok = <-resCh; ok {
		t.Fatalf("resCh is open, want closed")
	}
	if _, ok = <-errCh; ok {
		t.Fatalf("errCh is open, want closed")
	}
}

func TestASTBuilderInvalidDirective(t *testing.T) {
	var (
		jctx         = journal.NewContext()
		ctx          = context.Background()
		astBuilder   = ASTBuilder{Context: jctx}
		want         = &ast.AST{Context: jctx, Days: make(map[time.Time]*ast.Day)}
		inCh         = make(chan ast.Directive)
		resCh, errCh = astBuilder.BuildAST(ctx, inCh)
	)

	go func() {
		defer close(inCh)
		inCh <- DummyDirective{}
		inCh <- DummyDirective{}
	}()

	if err, ok := <-errCh; !ok || err == nil {
		t.Fatalf("<-errCh = %v, %t, want true, some error>", err, ok)
	}
	if err, ok := <-errCh; !ok || err == nil {
		t.Fatalf("<-errCh = %v, %t, want true, some error>", err, ok)
	}
	got, ok := <-resCh
	if !ok {
		t.Fatalf("ok = false, want ok = true")
	}
	if diff := cmp.Diff(got, want, cmpopts.IgnoreUnexported(journal.Context{}, journal.Commodity{}, journal.Account{})); diff != "" {
		t.Fatalf("unexpected diff (+got/-want):\n%s", diff)
	}
	if _, ok = <-resCh; ok {
		t.Fatalf("resCh is open, want closed")
	}
	if _, ok = <-errCh; ok {
		t.Fatalf("errCh is open, want closed")
	}
}

func TestASTBuilderCanceled(t *testing.T) {
	var (
		jctx         = journal.NewContext()
		ctx, cancel  = context.WithCancel(context.Background())
		astBuilder   = ASTBuilder{Context: jctx}
		inCh         = make(chan ast.Directive)
		resCh, errCh = astBuilder.BuildAST(ctx, inCh)
	)

	cancel()

	if _, ok := <-resCh; ok {
		t.Fatalf("resCh is open, want closed")
	}
	if _, ok := <-errCh; ok {
		t.Fatalf("errCh is open, want closed")
	}
}
