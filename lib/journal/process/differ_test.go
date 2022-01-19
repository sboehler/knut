package process

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/sboehler/knut/lib/common/amounts"
	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/val"
	"github.com/shopspring/decimal"
)

func TestDifferHappyCase(t *testing.T) {
	var (
		jctx   = journal.NewContext()
		td     = newTestData(jctx)
		differ = Differ{Diff: true}
		inCh   = make(chan *val.Day)
		ctx    = context.Background()
		day1   = &val.Day{
			Date: td.date1,
			Values: amounts.Amounts{
				amounts.CommodityAccount{Account: td.account1, Commodity: td.commodity1}: decimal.NewFromInt(10),
			},
		}
		day2 = &val.Day{
			Date: td.date2,
			Values: amounts.Amounts{
				amounts.CommodityAccount{Account: td.account1, Commodity: td.commodity1}: decimal.NewFromInt(7),
			},
		}
		day3 = &val.Day{
			Date: td.date3,
			Values: amounts.Amounts{
				amounts.CommodityAccount{Account: td.account1, Commodity: td.commodity1}: decimal.NewFromInt(8),
			},
		}
		want = []*val.Day{
			{
				Date: td.date1,
				Values: amounts.Amounts{
					amounts.CommodityAccount{Account: td.account1, Commodity: td.commodity1}: decimal.NewFromInt(10),
				},
			},
			{
				Date: td.date2,
				Values: amounts.Amounts{
					amounts.CommodityAccount{Account: td.account1, Commodity: td.commodity1}: decimal.NewFromInt(-3),
				},
			},
			{
				Date: td.date3,
				Values: amounts.Amounts{
					amounts.CommodityAccount{Account: td.account1, Commodity: td.commodity1}: decimal.NewFromInt(1),
				},
			},
		}
	)
	resCh, errCh := differ.ProcessStream(ctx, inCh)
	go func() {
		defer close(inCh)
		inCh <- day1
		inCh <- day2
		inCh <- day3
	}()
	var got []*val.Day
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
	if diff := cmp.Diff(want, got, cmpopts.IgnoreUnexported(journal.Context{}, journal.Commodity{}, journal.Account{})); diff != "" {
		t.Fatalf("unexpected diff (+got/-want):\n%s", diff)
	}
	if _, ok := <-resCh; ok {
		t.Fatalf("resCh is open, want closed")
	}
	if _, ok := <-errCh; ok {
		t.Fatalf("errCh is open, want closed")
	}
}

func TestDifferCanceled(t *testing.T) {
	var (
		ctx, cancel  = context.WithCancel(context.Background())
		astBuilder   = Differ{Diff: true}
		inCh         = make(chan *val.Day)
		resCh, errCh = astBuilder.ProcessStream(ctx, inCh)
	)

	cancel()

	if _, ok := <-resCh; ok {
		t.Fatalf("resCh is open, want closed")
	}
	if _, ok := <-errCh; ok {
		t.Fatalf("errCh is open, want closed")
	}
}
