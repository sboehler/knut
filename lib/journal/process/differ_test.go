package process

import (
	"context"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/sboehler/knut/lib/common/amounts"
	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/common/date"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
)

func TestDifferHappyCase(t *testing.T) {
	var (
		jctx   = journal.NewContext()
		td     = newTestData(jctx)
		differ = PeriodDiffer{}
		ctx    = context.Background()
		day1   = &ast.Period{
			Period: date.Period{End: td.date1},
			Days: []*ast.Day{
				{
					Transactions: []*ast.Transaction{td.trx1},
				},
			},
		}
		day2 = &ast.Period{
			Period: date.Period{End: td.date2},
			Days: []*ast.Day{
				{
					Transactions: []*ast.Transaction{td.trx2},
				},
			},
		}
		want = []*ast.Period{
			{
				Period: date.Period{End: td.date1},
				Amounts: amounts.Amounts{
					amounts.CommodityAccount{Account: td.trx1.Postings()[0].Credit, Commodity: td.trx1.Postings()[0].Commodity}: td.trx1.Postings()[0].Amount.Neg(),
					amounts.CommodityAccount{Account: td.trx1.Postings()[0].Debit, Commodity: td.trx1.Postings()[0].Commodity}:  td.trx1.Postings()[0].Amount,
				},
				Days: []*ast.Day{
					{
						Transactions: []*ast.Transaction{td.trx1},
					},
				},
			},
			{
				Period: date.Period{End: td.date2},
				Amounts: amounts.Amounts{
					amounts.CommodityAccount{Account: td.trx2.Postings()[0].Credit, Commodity: td.trx2.Postings()[0].Commodity}: td.trx2.Postings()[0].Amount.Neg(),
					amounts.CommodityAccount{Account: td.trx2.Postings()[0].Debit, Commodity: td.trx2.Postings()[0].Commodity}:  td.trx2.Postings()[0].Amount,
				},
				Days: []*ast.Day{
					{
						Transactions: []*ast.Transaction{td.trx2},
					},
				},
			},
		}
	)

	got, err := cpr.RunTestEngine[*ast.Period, *ast.Period](ctx, differ, day1, day2)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sort.Slice(got, func(i, j int) bool {
		return got[i].Period.End.Before(got[j].Period.End)
	})

	if diff := cmp.Diff(want, got, cmp.AllowUnexported(ast.Transaction{}), cmpopts.IgnoreUnexported(journal.Context{}, journal.Commodity{}, journal.Account{})); diff != "" {
		t.Fatalf("unexpected diff (+got/-want):\n%s", diff)
	}
}
