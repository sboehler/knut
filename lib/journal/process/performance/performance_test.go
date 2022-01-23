package performance

import (
	"regexp"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
	"github.com/sboehler/knut/lib/journal/val"
	"github.com/shopspring/decimal"
)

func TestComputeFlows(t *testing.T) {
	var (
		ctx          = journal.NewContext()
		chf, _       = ctx.GetCommodity("CHF")
		usd, _       = ctx.GetCommodity("USD")
		gbp, _       = ctx.GetCommodity("GBP")
		aapl, _      = ctx.GetCommodity("AAPL")
		portfolio, _ = ctx.GetAccount("Assets:Portfolio")
		acc1, _      = ctx.GetAccount("Assets:Acc1")
		acc2, _      = ctx.GetAccount("Assets:Acc2")
		dividend, _  = ctx.GetAccount("Income:Dividends")
		expense, _   = ctx.GetAccount("Expenses:Investments")
		equity, _    = ctx.GetAccount("Equity:Equity")
	)
	chf.IsCurrency = true
	usd.IsCurrency = true
	gbp.IsCurrency = true

	var (
		tests = []struct {
			desc string
			trx  *ast.Transaction
			want *PerfPeriod
		}{
			{
				desc: "outflow",
				trx: &ast.Transaction{
					Postings: []ast.Posting{
						{
							Credit:    portfolio,
							Debit:     acc2,
							Amount:    decimal.NewFromInt(1),
							Commodity: usd,
						},
					},
				},
				want: &PerfPeriod{Outflow: pcv{usd: -1.0}},
			},
			{
				desc: "inflow",
				trx: &ast.Transaction{
					Postings: []ast.Posting{
						{
							Credit:    acc1,
							Debit:     portfolio,
							Amount:    decimal.NewFromInt(1),
							Commodity: usd,
						},
					},
				},
				want: &PerfPeriod{Inflow: pcv{usd: 1.0}},
			},
			{
				desc: "dividend",
				trx: &ast.Transaction{
					Postings: []ast.Posting{
						{
							Credit:    dividend,
							Debit:     portfolio,
							Amount:    decimal.NewFromInt(1),
							Commodity: usd,
							Targets:   []*journal.Commodity{aapl, usd},
						},
					},
				},
				want: &PerfPeriod{
					InternalInflow:  pcv{usd: 1.0},
					InternalOutflow: pcv{aapl: -1.0},
				},
			},
			{
				desc: "expense",
				trx: &ast.Transaction{
					Postings: []ast.Posting{
						{
							Credit:    portfolio,
							Debit:     expense,
							Amount:    decimal.NewFromInt(1),
							Commodity: usd,
							Targets:   []*journal.Commodity{aapl, usd},
						},
					},
				},
				want: &PerfPeriod{
					InternalInflow:  pcv{aapl: 1.0},
					InternalOutflow: pcv{usd: -1.0},
				},
			},
			{
				desc: "expense with effect on porfolio",
				trx: &ast.Transaction{
					Postings: []ast.Posting{
						{
							Credit:    portfolio,
							Debit:     expense,
							Amount:    decimal.NewFromInt(1),
							Commodity: usd,
							Targets:   make([]*journal.Commodity, 0),
						},
					},
				},
				want: &PerfPeriod{
					InternalOutflow: pcv{usd: -1.0},
					PortfolioInflow: 1.0,
				},
			},
			{
				desc: "stock purchase",
				trx: &ast.Transaction{
					Postings: []ast.Posting{
						{
							Credit:    portfolio,
							Debit:     equity,
							Amount:    decimal.NewFromInt(1010),
							Commodity: usd,
							Targets:   []*journal.Commodity{usd, aapl},
						},
						{
							Credit:    equity,
							Debit:     portfolio,
							Amount:    decimal.NewFromInt(1000),
							Commodity: aapl,
							Targets:   []*journal.Commodity{usd, aapl},
						},
					},
				},
				want: &PerfPeriod{
					InternalInflow:  pcv{aapl: 1010.0},
					InternalOutflow: pcv{usd: -1010.0},
				},
			},
			{
				desc: "stock purchase with fee",
				trx: &ast.Transaction{
					Postings: []ast.Posting{
						{
							Credit:    portfolio,
							Debit:     equity,
							Amount:    decimal.NewFromInt(1010),
							Commodity: usd,
							Targets:   []*journal.Commodity{usd, aapl},
						},
						{
							Credit:    equity,
							Debit:     portfolio,
							Amount:    decimal.NewFromInt(1000),
							Commodity: aapl,
							Targets:   []*journal.Commodity{usd, aapl},
						},
						{
							Credit:    portfolio,
							Debit:     equity,
							Amount:    decimal.NewFromInt(10),
							Commodity: usd,
							Targets:   []*journal.Commodity{usd, aapl},
						},
					},
				},
				want: &PerfPeriod{
					InternalInflow:  pcv{aapl: 1020.0},
					InternalOutflow: pcv{usd: -1020.0},
				},
			},
			{
				desc: "stock sale",
				trx: &ast.Transaction{
					Postings: []ast.Posting{
						{
							Credit:    portfolio,
							Debit:     equity,
							Amount:    decimal.NewFromInt(1000),
							Commodity: aapl,
							Targets:   []*journal.Commodity{usd, aapl},
						},
						{
							Credit:    equity,
							Debit:     portfolio,
							Amount:    decimal.NewFromInt(990),
							Commodity: usd,
							Targets:   []*journal.Commodity{usd, aapl},
						},
					},
				},
				want: &PerfPeriod{
					InternalInflow:  pcv{usd: 990.0},
					InternalOutflow: pcv{aapl: -990.0},
				},
			},

			{
				desc: "forex without fee",
				trx: &ast.Transaction{
					Postings: []ast.Posting{
						{
							Credit:    portfolio,
							Debit:     equity,
							Amount:    decimal.NewFromInt(1400),
							Commodity: gbp,
							Targets:   []*journal.Commodity{usd, gbp},
						},
						{
							Credit:    equity,
							Debit:     portfolio,
							Amount:    decimal.NewFromInt(1350),
							Commodity: usd,
							Targets:   []*journal.Commodity{usd, gbp},
						},
					},
				},
				want: &PerfPeriod{
					InternalOutflow: pcv{gbp: -1375.0},
					InternalInflow:  pcv{usd: 1375.0},
				},
			},
			{
				desc: "forex with fee",
				trx: &ast.Transaction{
					Postings: []ast.Posting{
						{
							Credit:    portfolio,
							Debit:     equity,
							Amount:    decimal.NewFromInt(1400),
							Commodity: gbp,
							Targets:   []*journal.Commodity{usd, gbp},
						},
						{
							Credit:    equity,
							Debit:     portfolio,
							Amount:    decimal.NewFromInt(1350),
							Commodity: usd,
							Targets:   []*journal.Commodity{usd, gbp},
						},
						{
							Credit:    portfolio,
							Debit:     expense,
							Amount:    decimal.NewFromInt(10),
							Commodity: chf,
							Targets:   []*journal.Commodity{usd, gbp},
						},
					},
				},
				want: &PerfPeriod{
					InternalOutflow: pcv{gbp: -1370.0, chf: -10},
					InternalInflow:  pcv{usd: 1380.0},
				},
			},
			{
				desc: "forex with native fee",
				trx: &ast.Transaction{
					Postings: []ast.Posting{
						{
							Credit:    portfolio,
							Debit:     equity,
							Amount:    decimal.NewFromInt(1400),
							Commodity: gbp,
							Targets:   []*journal.Commodity{usd, gbp},
						},
						{
							Credit:    equity,
							Debit:     portfolio,
							Amount:    decimal.NewFromInt(1350),
							Commodity: usd,
							Targets:   []*journal.Commodity{usd, gbp},
						},
						{
							Credit:    portfolio,
							Debit:     expense,
							Amount:    decimal.NewFromInt(10),
							Commodity: usd,
							Targets:   []*journal.Commodity{usd, gbp},
						},
					},
				},
				want: &PerfPeriod{
					InternalOutflow: pcv{gbp: -1370.0},
					InternalInflow:  pcv{usd: 1370.0},
				},
			},
		}
	)
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			var (
				d = &val.Day{
					Date:         time.Date(2021, 11, 15, 0, 0, 0, 0, time.UTC),
					Transactions: []*ast.Transaction{test.trx},
				}

				fc = Calculator{
					Filter:    journal.Filter{Accounts: regexp.MustCompile("Assets:Portfolio")},
					Valuation: chf,
				}
			)

			got := fc.computeFlows(d)

			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Fatalf("unexpected diff (-want, +got):\n%s", diff)
			}
		})
	}

}
