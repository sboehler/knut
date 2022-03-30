package process

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/sboehler/knut/lib/common/amounts"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
	"github.com/shopspring/decimal"
)

func TestPASTBuilderHappyCase(t *testing.T) {
	var (
		jctx        = journal.NewContext()
		td          = newTestData(jctx)
		pastBuilder = PASTBuilder{Context: jctx}
		ctx         = context.Background()
		input       = []*ast.Day{
			{
				Date:         td.date1,
				Openings:     []*ast.Open{td.open1, td.open2},
				Prices:       []*ast.Price{td.price1},
				Transactions: []*ast.Transaction{td.trx1},
			}, {
				Date:         td.date2,
				Transactions: []*ast.Transaction{td.trx2},
			},
		}
		want = []*ast.Day{
			{
				Date:         td.date1,
				Openings:     []*ast.Open{td.open1, td.open2},
				Prices:       []*ast.Price{td.price1},
				Transactions: []*ast.Transaction{td.trx1},
				Amounts: amounts.Amounts{
					amounts.CommodityAccount{Account: td.account1, Commodity: td.commodity1}: decimal.NewFromInt(-10),
					amounts.CommodityAccount{Account: td.account2, Commodity: td.commodity1}: decimal.NewFromInt(10),
				},
			},
			{
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

	got, err := ast.RunTestEngine[*ast.Day](ctx, &pastBuilder, input...)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if diff := cmp.Diff(got, want, cmpopts.IgnoreUnexported(journal.Context{}, journal.Commodity{}, journal.Account{})); diff != "" {
		t.Fatalf("unexpected diff (+got/-want):\n%s", diff)
	}
}
