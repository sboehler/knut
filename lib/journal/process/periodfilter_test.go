package process

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/sboehler/knut/lib/common/amounts"
	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/common/date"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
	"github.com/shopspring/decimal"
)

func datedTrx(y int, m time.Month, d int) *ast.Transaction {
	return ast.TransactionBuilder{Date: date.Date(y, m, d)}.Build()
}

func TestPeriodFilter(t *testing.T) {
	var (
		jctx = journal.NewContext()
		td   = newTestData(jctx)
	)
	day := func(y int, m time.Month, d int, v int64, trx ...*ast.Transaction) *ast.Day {
		return &ast.Day{
			Date:         date.Date(y, m, d),
			Transactions: trx,
			Value: amounts.Amounts{
				amounts.CommodityAccount{Account: td.account1, Commodity: td.commodity1}: decimal.NewFromInt(v),
			},
		}
	}
	period := func(start, end time.Time, v int64, days ...*ast.Day) *ast.Period {
		return &ast.Period{
			Period: date.Period{Start: start, End: end},
			Days:   days,
			Values: amounts.Amounts{
				amounts.CommodityAccount{Account: td.account1, Commodity: td.commodity1}: decimal.NewFromInt(v),
			},
		}
	}

	var (
		tests = []struct {
			desc    string
			sut     PeriodFilter
			input   []*ast.Day
			want    []*ast.Period
			wantErr bool
		}{
			{
				desc: "no input",
				sut:  PeriodFilter{},
			},
			{
				desc: "no period, no from date",
				sut: PeriodFilter{
					To: date.Date(2022, 1, 10),
				},
				input: []*ast.Day{
					day(2022, 1, 2, 1, datedTrx(2022, 1, 2)),
					day(2022, 1, 3, 2),
					day(2022, 1, 4, 3),
				},
				want: []*ast.Period{
					period(date.Date(2022, 1, 2), date.Date(2022, 1, 10), 3,
						day(2022, 1, 2, 1, datedTrx(2022, 1, 2)),
						day(2022, 1, 3, 2),
						day(2022, 1, 4, 3),
					),
				},
			},
			{
				desc: "monthly, no from date",
				sut: PeriodFilter{
					To:       date.Date(2022, 1, 10),
					Interval: date.Monthly,
				},
				input: []*ast.Day{
					day(2022, 1, 2, 100, datedTrx(2022, 1, 2)),
					day(2022, 1, 3, 200),
					day(2022, 1, 4, 300),
				},
				want: []*ast.Period{
					period(date.Date(2022, 1, 1), date.Date(2022, 1, 10), 300,
						day(2022, 1, 2, 100, datedTrx(2022, 1, 2)),
						day(2022, 1, 3, 200),
						day(2022, 1, 4, 300)),
				},
			},
			{
				desc: "monthly, last 5, no from date",
				sut: PeriodFilter{
					To:       date.Date(2022, 1, 10),
					Interval: date.Monthly,
					Last:     5,
				},
				input: []*ast.Day{
					day(2021, 1, 1, 100, datedTrx(2021, 1, 1)),
					day(2022, 1, 1, 200, datedTrx(2022, 1, 1)),
					day(2022, 1, 4, 300, datedTrx(2022, 1, 4)),
				},
				want: []*ast.Period{
					period(date.Date(2021, 9, 1), date.Date(2021, 9, 30), 100),
					period(date.Date(2021, 10, 1), date.Date(2021, 10, 31), 100),
					period(date.Date(2021, 11, 1), date.Date(2021, 11, 30), 100),
					period(date.Date(2021, 12, 1), date.Date(2021, 12, 31), 100),
					period(date.Date(2022, 1, 1), date.Date(2022, 1, 10), 300,
						day(2022, 1, 1, 200, datedTrx(2022, 1, 1)),
						day(2022, 1, 4, 300, datedTrx(2022, 1, 4))),
				},
			},
		}
	)

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {

			ctx := context.Background()

			got, err := cpr.RunTestEngine[*ast.Day, *ast.Period](ctx, test.sut, test.input...)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if diff := cmp.Diff(test.want, got, cmp.AllowUnexported(ast.Transaction{}), cmpopts.IgnoreUnexported(journal.Context{}, journal.Commodity{}, journal.Account{})); diff != "" {
				t.Fatalf("unexpected diff (+got/-want):\n%s", diff)
			}
		})
	}
}
