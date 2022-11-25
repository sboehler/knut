package performance

import (
	"regexp"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/sboehler/knut/lib/common/filter"
	"github.com/sboehler/knut/lib/common/slice"
	"github.com/sboehler/knut/lib/journal"
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
			trx  *journal.Transaction
			want *journal.Performance
		}{
			{
				desc: "outflow",
				trx: journal.TransactionBuilder{
					Postings: journal.PostingBuilder{
						Credit:    portfolio,
						Debit:     acc2,
						Amount:    decimal.NewFromInt(1),
						Commodity: usd,
					}.Singleton(),
				}.Build(),
				want: &journal.Performance{Outflow: pcv{usd: -1.0}},
			},
			{
				desc: "inflow",
				trx: journal.TransactionBuilder{
					Postings: journal.PostingBuilder{
						Credit:    acc1,
						Debit:     portfolio,
						Amount:    decimal.NewFromInt(1),
						Commodity: usd,
					}.Singleton(),
				}.Build(),
				want: &journal.Performance{Inflow: pcv{usd: 1.0}},
			},
			{
				desc: "dividend",
				trx: journal.TransactionBuilder{
					Postings: journal.PostingBuilder{
						Credit:    dividend,
						Debit:     portfolio,
						Amount:    decimal.NewFromInt(1),
						Commodity: usd,
						Targets:   []*journal.Commodity{aapl, usd},
					}.Singleton(),
				}.Build(),
				want: &journal.Performance{
					InternalInflow:  pcv{usd: 1.0},
					InternalOutflow: pcv{aapl: -1.0},
				},
			},
			{
				desc: "expense",
				trx: journal.TransactionBuilder{
					Postings: journal.PostingBuilder{
						Credit:    portfolio,
						Debit:     expense,
						Amount:    decimal.NewFromInt(1),
						Commodity: usd,
						Targets:   []*journal.Commodity{aapl, usd},
					}.Singleton(),
				}.Build(),
				want: &journal.Performance{
					InternalInflow:  pcv{aapl: 1.0},
					InternalOutflow: pcv{usd: -1.0},
				},
			},
			{
				desc: "expense with effect on porfolio",
				trx: journal.TransactionBuilder{
					Postings: journal.PostingBuilder{
						Credit:    portfolio,
						Debit:     expense,
						Amount:    decimal.NewFromInt(1),
						Commodity: usd,
						Targets:   make([]*journal.Commodity, 0),
					}.Singleton(),
				}.Build(),
				want: &journal.Performance{
					InternalOutflow: pcv{usd: -1.0},
					PortfolioInflow: 1.0,
				},
			},
			{
				desc: "stock purchase",
				trx: journal.TransactionBuilder{

					Postings: slice.Concat(
						journal.PostingBuilder{
							Credit:    portfolio,
							Debit:     equity,
							Amount:    decimal.NewFromInt(1010),
							Commodity: usd,
							Targets:   []*journal.Commodity{usd, aapl},
						}.Build(),
						journal.PostingBuilder{
							Credit:    equity,
							Debit:     portfolio,
							Amount:    decimal.NewFromInt(1000),
							Commodity: aapl,
							Targets:   []*journal.Commodity{usd, aapl},
						}.Build(),
					),
				}.Build(),
				want: &journal.Performance{
					InternalInflow:  pcv{aapl: 1010.0},
					InternalOutflow: pcv{usd: -1010.0},
				},
			},
			{
				desc: "stock purchase with fee",
				trx: journal.TransactionBuilder{
					Postings: slice.Concat(
						journal.PostingBuilder{
							Credit:    portfolio,
							Debit:     equity,
							Amount:    decimal.NewFromInt(1010),
							Commodity: usd,
							Targets:   []*journal.Commodity{usd, aapl},
						}.Build(),
						journal.PostingBuilder{
							Credit:    equity,
							Debit:     portfolio,
							Amount:    decimal.NewFromInt(1000),
							Commodity: aapl,
							Targets:   []*journal.Commodity{usd, aapl},
						}.Build(),
						journal.PostingBuilder{
							Credit:    portfolio,
							Debit:     equity,
							Amount:    decimal.NewFromInt(10),
							Commodity: usd,
							Targets:   []*journal.Commodity{usd, aapl},
						}.Build(),
					),
				}.Build(),
				want: &journal.Performance{
					InternalInflow:  pcv{aapl: 1020.0},
					InternalOutflow: pcv{usd: -1020.0},
				},
			},
			{
				desc: "stock sale",
				trx: journal.TransactionBuilder{
					Postings: slice.Concat(
						journal.PostingBuilder{
							Credit:    portfolio,
							Debit:     equity,
							Amount:    decimal.NewFromInt(1000),
							Commodity: aapl,
							Targets:   []*journal.Commodity{usd, aapl},
						}.Build(),
						journal.PostingBuilder{
							Credit:    equity,
							Debit:     portfolio,
							Amount:    decimal.NewFromInt(990),
							Commodity: usd,
							Targets:   []*journal.Commodity{usd, aapl},
						}.Build(),
					),
				}.Build(),
				want: &journal.Performance{
					InternalInflow:  pcv{usd: 990.0},
					InternalOutflow: pcv{aapl: -990.0},
				},
			},

			{
				desc: "forex without fee",
				trx: journal.TransactionBuilder{
					Postings: slice.Concat(
						journal.PostingBuilder{
							Credit:    portfolio,
							Debit:     equity,
							Amount:    decimal.NewFromInt(1400),
							Commodity: gbp,
							Targets:   []*journal.Commodity{usd, gbp},
						}.Build(),
						journal.PostingBuilder{
							Credit:    equity,
							Debit:     portfolio,
							Amount:    decimal.NewFromInt(1350),
							Commodity: usd,
							Targets:   []*journal.Commodity{usd, gbp},
						}.Build(),
					),
				}.Build(),
				want: &journal.Performance{
					InternalOutflow: pcv{gbp: -1375.0},
					InternalInflow:  pcv{usd: 1375.0},
				},
			},
			{
				desc: "forex with fee",
				trx: journal.TransactionBuilder{
					Postings: slice.Concat(
						journal.PostingBuilder{
							Credit:    portfolio,
							Debit:     equity,
							Amount:    decimal.NewFromInt(1400),
							Commodity: gbp,
							Targets:   []*journal.Commodity{usd, gbp},
						}.Build(),
						journal.PostingBuilder{
							Credit:    equity,
							Debit:     portfolio,
							Amount:    decimal.NewFromInt(1350),
							Commodity: usd,
							Targets:   []*journal.Commodity{usd, gbp},
						}.Build(),
						journal.PostingBuilder{
							Credit:    portfolio,
							Debit:     expense,
							Amount:    decimal.NewFromInt(10),
							Commodity: chf,
							Targets:   []*journal.Commodity{usd, gbp},
						}.Build(),
					),
				}.Build(),
				want: &journal.Performance{
					InternalOutflow: pcv{gbp: -1370.0, chf: -10},
					InternalInflow:  pcv{usd: 1380.0},
				},
			},
			{
				desc: "forex with native fee",
				trx: journal.TransactionBuilder{
					Postings: slice.Concat(
						journal.PostingBuilder{
							Credit:    portfolio,
							Debit:     equity,
							Amount:    decimal.NewFromInt(1400),
							Commodity: gbp,
							Targets:   []*journal.Commodity{usd, gbp},
						}.Build(),
						journal.PostingBuilder{
							Credit:    equity,
							Debit:     portfolio,
							Amount:    decimal.NewFromInt(1350),
							Commodity: usd,
							Targets:   []*journal.Commodity{usd, gbp},
						}.Build(),
						journal.PostingBuilder{
							Credit:    portfolio,
							Debit:     expense,
							Amount:    decimal.NewFromInt(10),
							Commodity: usd,
							Targets:   []*journal.Commodity{usd, gbp},
						}.Build(),
					),
				}.Build(),
				want: &journal.Performance{
					InternalOutflow: pcv{gbp: -1370.0},
					InternalInflow:  pcv{usd: 1370.0},
				},
			},
		}
	)
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			var (
				d = &journal.Day{
					Date:         time.Date(2021, 11, 15, 0, 0, 0, 0, time.UTC),
					Transactions: []*journal.Transaction{test.trx},
				}

				fc = Calculator{
					AccountFilter: filter.ByName[*journal.Account]([]*regexp.Regexp{
						regexp.MustCompile("Assets:Portfolio")}),
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
