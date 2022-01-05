package process

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/common/date"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
	"github.com/sboehler/knut/lib/journal/val"
)

func TestPeriodFilter(t *testing.T) {

	day := func(y int, m time.Month, d int, marker ...bool) *val.Day {
		var trx []*ast.Transaction
		if len(marker) > 0 {
			trx = append(trx, nil)
		}
		return &val.Day{Date: date.Date(y, m, d), Transactions: trx}
	}

	var (
		// jctx = journal.NewContext()
		// td   = newTestData(jctx)

		tests = []struct {
			desc    string
			sut     PeriodFilter
			input   func(chan *val.Day)
			want    []*val.Day
			wantErr bool
		}{
			{
				desc:  "no input",
				sut:   PeriodFilter{},
				input: func(ch chan *val.Day) {},
			},
			{
				desc: "no period, no from date",
				sut: PeriodFilter{
					To: date.Date(2022, 1, 10),
				},
				input: func(ch chan *val.Day) {
					ch <- day(2022, 1, 2, true)
					ch <- day(2022, 1, 3)
					ch <- day(2022, 1, 4)
				},
				want: []*val.Day{day(2022, 1, 2), day(2022, 1, 10)},
			},
			{
				desc: "monthly, no from date",
				sut: PeriodFilter{
					To:     date.Date(2022, 1, 10),
					Period: date.Monthly,
				},
				input: func(ch chan *val.Day) {
					ch <- day(2022, 1, 2, true)
					ch <- day(2022, 1, 3)
					ch <- day(2022, 1, 4)
				},
				want: []*val.Day{day(2021, 12, 31), day(2022, 1, 31)},
			},
			{
				desc: "monthly, last 5, no from date",
				sut: PeriodFilter{
					To:     date.Date(2022, 1, 10),
					Period: date.Monthly,
					Last:   5,
				},
				input: func(ch chan *val.Day) {
					ch <- day(2021, 1, 2, true)
					ch <- day(2022, 1, 3)
					ch <- day(2022, 1, 4)
				},
				want: []*val.Day{
					day(2021, 9, 30),
					day(2021, 10, 31),
					day(2021, 11, 30),
					day(2021, 12, 31),
					day(2022, 1, 31)},
			},
		}
	)

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			var (
				ctx  = context.Background()
				inCh = make(chan *val.Day)
			)
			resCh, errCh := test.sut.ProcessStream(ctx, inCh)

			go func() {
				defer close(inCh)
				test.input(inCh)
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

			if diff := cmp.Diff(got, test.want, cmpopts.IgnoreUnexported(journal.Context{}, journal.Commodity{}, journal.Account{})); diff != "" {
				t.Fatalf("unexpected diff (+got/-want):\n%s", diff)
			}

			if _, ok := <-resCh; ok {
				t.Fatalf("resCh is open, want closed")
			}
			if _, ok := <-errCh; ok {
				t.Fatalf("errCh is open, want closed")
			}

		})
	}
}