package process

import (
	"context"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/sboehler/knut/lib/common/amounts"
	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
)

func TestDifferHappyCase(t *testing.T) {
	var (
		jctx   = journal.NewContext()
		td     = newTestData(jctx)
		differ = PeriodDiffer{}
		ctx    = context.Background()
		day1   = &ast.Day{
			Date: td.date1,
			PeriodDays: []*ast.Day{
				{
					Transactions: []*ast.Transaction{td.trx1},
				},
			},
		}
		day2 = &ast.Day{
			Date: td.date2,
			PeriodDays: []*ast.Day{
				{
					Transactions: []*ast.Transaction{td.trx2},
				},
			},
		}
		want = []*ast.Day{
			{
				Date: td.date1,
				Amounts: amounts.Amounts{
					amounts.CommodityAccount{Account: td.trx1.Postings()[0].Credit, Commodity: td.trx1.Postings()[0].Commodity}: td.trx1.Postings()[0].Amount.Neg(),
					amounts.CommodityAccount{Account: td.trx1.Postings()[0].Debit, Commodity: td.trx1.Postings()[0].Commodity}:  td.trx1.Postings()[0].Amount,
				},
				PeriodDays: []*ast.Day{
					{
						Transactions: []*ast.Transaction{td.trx1},
					},
				},
			},
			{
				Date: td.date2,
				Amounts: amounts.Amounts{
					amounts.CommodityAccount{Account: td.trx2.Postings()[0].Credit, Commodity: td.trx2.Postings()[0].Commodity}: td.trx2.Postings()[0].Amount.Neg(),
					amounts.CommodityAccount{Account: td.trx2.Postings()[0].Debit, Commodity: td.trx2.Postings()[0].Commodity}:  td.trx2.Postings()[0].Amount,
				},
				PeriodDays: []*ast.Day{
					{
						Transactions: []*ast.Transaction{td.trx2},
					},
				},
			},
		}
	)

	got, err := cpr.RunTestEngine[*ast.Day](ctx, differ, day1, day2)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sort.Slice(got, func(i, j int) bool {
		return got[i].Date.Before(got[j].Date)
	})

	if diff := cmp.Diff(want, got, cmp.AllowUnexported(ast.Transaction{}), cmpopts.IgnoreUnexported(journal.Context{}, journal.Commodity{}, journal.Account{})); diff != "" {
		t.Fatalf("unexpected diff (+got/-want):\n%s", diff)
	}
}
