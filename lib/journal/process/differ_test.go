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
	"github.com/shopspring/decimal"
)

func TestDifferHappyCase(t *testing.T) {
	var (
		jctx   = journal.NewContext()
		td     = newTestData(jctx)
		differ = PeriodDiffer{}
		ctx    = context.Background()
		day1   = &ast.Period{
			Period: date.Period{End: td.date1},
			Amounts: amounts.Amounts{
				amounts.AccountCommodityKey(td.account1, td.commodity1): decimal.NewFromInt(100),
				amounts.AccountCommodityKey(td.account2, td.commodity2): decimal.NewFromInt(200),
			},
			PrevAmounts: amounts.Amounts{
				amounts.AccountCommodityKey(td.account1, td.commodity1): decimal.NewFromInt(20),
				amounts.AccountCommodityKey(td.account2, td.commodity2): decimal.NewFromInt(20),
			},
		}
		ca1  = amounts.AccountCommodityKey(td.account1, td.commodity1)
		ca2  = amounts.AccountCommodityKey(td.account2, td.commodity2)
		day2 = &ast.Period{
			Period: date.Period{End: td.date2},
			Values: amounts.Amounts{
				ca1: decimal.NewFromInt(100),
				ca2: decimal.NewFromInt(200),
			},
			PrevValues: amounts.Amounts{
				ca1: decimal.NewFromInt(20),
				ca2: decimal.NewFromInt(20),
			},
		}
		want = []*ast.Period{
			{
				Period: date.Period{End: td.date1},
				Amounts: amounts.Amounts{
					ca1: decimal.NewFromInt(100),
					ca2: decimal.NewFromInt(200),
				},
				PrevAmounts: amounts.Amounts{
					ca1: decimal.NewFromInt(20),
					ca2: decimal.NewFromInt(20),
				},
				DeltaAmounts: amounts.Amounts{
					ca1: decimal.NewFromInt(80),
					ca2: decimal.NewFromInt(180),
				},
			},
			{
				Period: date.Period{End: td.date2},
				Values: amounts.Amounts{
					amounts.AccountCommodityKey(td.account1, td.commodity1): decimal.NewFromInt(100),
					amounts.AccountCommodityKey(td.account2, td.commodity2): decimal.NewFromInt(200),
				},
				PrevValues: amounts.Amounts{
					amounts.AccountCommodityKey(td.account1, td.commodity1): decimal.NewFromInt(20),
					amounts.AccountCommodityKey(td.account2, td.commodity2): decimal.NewFromInt(20),
				},
				DeltaValues: amounts.Amounts{
					amounts.AccountCommodityKey(td.account1, td.commodity1): decimal.NewFromInt(80),
					amounts.AccountCommodityKey(td.account2, td.commodity2): decimal.NewFromInt(180),
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
