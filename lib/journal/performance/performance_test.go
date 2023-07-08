package performance

import (
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sboehler/knut/lib/common/date"
	"github.com/sboehler/knut/lib/common/filter"
	"github.com/sboehler/knut/lib/journal"
	"github.com/shopspring/decimal"
)

func TestComputeFlows(t *testing.T) {

	ctx := journal.NewContext()
	chf := ctx.Commodity("CHF")
	usd := ctx.Commodity("USD")
	gbp := ctx.Commodity("GBP")
	aapl := ctx.Commodity("AAPL")
	portfolio := ctx.Account("Assets:Portfolio")
	acc1 := ctx.Account("Assets:Acc1")
	acc2 := ctx.Account("Assets:Acc2")
	dividend := ctx.Account("Income:Dividends")
	expense := ctx.Account("Expenses:Investments")
	equity := ctx.Account("Equity:Equity")

	chf.IsCurrency = true
	usd.IsCurrency = true
	gbp.IsCurrency = true

	tests := []struct {
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
				}.Build(),
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
				}.Build(),
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
				}.Build(),
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
				}.Build(),
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
				}.Build(),
			}.Build(),
			want: &journal.Performance{
				InternalOutflow: pcv{usd: -1.0},
				PortfolioInflow: 1.0,
			},
		},
		{
			desc: "stock purchase",
			trx: journal.TransactionBuilder{

				Postings: journal.PostingBuilders{
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
				}.Build(),
			}.Build(),
			want: &journal.Performance{
				InternalInflow:  pcv{aapl: 1010.0},
				InternalOutflow: pcv{usd: -1010.0},
			},
		},
		{
			desc: "stock purchase with fee",
			trx: journal.TransactionBuilder{
				Postings: journal.PostingBuilders{
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
				}.Build(),
			}.Build(),
			want: &journal.Performance{
				InternalInflow:  pcv{aapl: 1020.0},
				InternalOutflow: pcv{usd: -1020.0},
			},
		},
		{
			desc: "stock sale",
			trx: journal.TransactionBuilder{
				Postings: journal.PostingBuilders{
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
				}.Build(),
			}.Build(),
			want: &journal.Performance{
				InternalInflow:  pcv{usd: 990.0},
				InternalOutflow: pcv{aapl: -990.0},
			},
		},

		{
			desc: "forex without fee",
			trx: journal.TransactionBuilder{
				Postings: journal.PostingBuilders{
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
				}.Build(),
			}.Build(),
			want: &journal.Performance{
				InternalOutflow: pcv{gbp: -1375.0},
				InternalInflow:  pcv{usd: 1375.0},
			},
		},
		{
			desc: "forex with fee",
			trx: journal.TransactionBuilder{
				Postings: journal.PostingBuilders{
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
				}.Build(),
			}.Build(),
			want: &journal.Performance{
				InternalOutflow: pcv{gbp: -1370.0, chf: -10},
				InternalInflow:  pcv{usd: 1380.0},
			},
		},
		{
			desc: "forex with native fee",
			trx: journal.TransactionBuilder{
				Postings: journal.PostingBuilders{
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
				}.Build(),
			}.Build(),
			want: &journal.Performance{
				InternalOutflow: pcv{gbp: -1370.0},
				InternalInflow:  pcv{usd: 1370.0},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			day := &journal.Day{
				Date:         date.Date(2021, 11, 15),
				Transactions: []*journal.Transaction{test.trx},
			}
			calc := Calculator{
				AccountFilter: filter.ByName[*journal.Account]([]*regexp.Regexp{
					regexp.MustCompile("Assets:Portfolio"),
				}),
				Valuation: chf,
			}

			calc.ComputeFlows()(day)

			if diff := cmp.Diff(test.want, day.Performance); diff != "" {
				t.Fatalf("unexpected diff (-want, +got):\n%s", diff)
			}
		})
	}

}
