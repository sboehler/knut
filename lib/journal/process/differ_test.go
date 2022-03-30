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

func TestDifferHappyCase(t *testing.T) {
	var (
		jctx   = journal.NewContext()
		td     = newTestData(jctx)
		differ = Differ{Diff: true}
		ctx    = context.Background()
		day1   = &ast.Day{
			Date: td.date1,
			Value: amounts.Amounts{
				amounts.CommodityAccount{Account: td.account1, Commodity: td.commodity1}: decimal.NewFromInt(10),
			},
		}
		day2 = &ast.Day{
			Date: td.date2,
			Value: amounts.Amounts{
				amounts.CommodityAccount{Account: td.account1, Commodity: td.commodity1}: decimal.NewFromInt(7),
			},
		}
		day3 = &ast.Day{
			Date: td.date3,
			Value: amounts.Amounts{
				amounts.CommodityAccount{Account: td.account1, Commodity: td.commodity1}: decimal.NewFromInt(8),
			},
		}
		want = []*ast.Day{
			{
				Date: td.date1,
				Value: amounts.Amounts{
					amounts.CommodityAccount{Account: td.account1, Commodity: td.commodity1}: decimal.NewFromInt(10),
				},
			},
			{
				Date: td.date2,
				Value: amounts.Amounts{
					amounts.CommodityAccount{Account: td.account1, Commodity: td.commodity1}: decimal.NewFromInt(-3),
				},
			},
			{
				Date: td.date3,
				Value: amounts.Amounts{
					amounts.CommodityAccount{Account: td.account1, Commodity: td.commodity1}: decimal.NewFromInt(1),
				},
			},
		}
	)

	got, err := ast.RunTestEngine[*ast.Day](ctx, differ, day1, day2, day3)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if diff := cmp.Diff(want, got, cmpopts.IgnoreUnexported(journal.Context{}, journal.Commodity{}, journal.Account{})); diff != "" {
		t.Fatalf("unexpected diff (+got/-want):\n%s", diff)
	}
}
