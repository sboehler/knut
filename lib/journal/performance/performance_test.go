package performance

import (
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sboehler/knut/lib/common/date"
	"github.com/sboehler/knut/lib/common/predicate"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/model"
	"github.com/sboehler/knut/lib/model/posting"
	"github.com/sboehler/knut/lib/model/registry"
	"github.com/sboehler/knut/lib/model/transaction"
	"github.com/shopspring/decimal"
)

func TestComputeFlows(t *testing.T) {

	ctx := registry.New()
	chf := ctx.Commodities().MustGet("CHF")
	usd := ctx.Commodities().MustGet("USD")
	gbp := ctx.Commodities().MustGet("GBP")
	aapl := ctx.Commodities().MustGet("AAPL")
	portfolio := ctx.Accounts().MustGet("Assets:Portfolio")
	acc1 := ctx.Accounts().MustGet("Assets:Acc1")
	acc2 := ctx.Accounts().MustGet("Assets:Acc2")
	dividend := ctx.Accounts().MustGet("Income:Dividends")
	expense := ctx.Accounts().MustGet("Expenses:Investments")
	equity := ctx.Accounts().MustGet("Equity:Equity")

	chf.IsCurrency = true
	usd.IsCurrency = true
	gbp.IsCurrency = true

	tests := []struct {
		desc string
		trx  *model.Transaction
		want *journal.Performance
	}{
		{
			desc: "outflow",
			trx: transaction.Builder{
				Postings: posting.Builder{
					Credit:    portfolio,
					Debit:     acc2,
					Value:     decimal.NewFromInt(1),
					Commodity: usd,
				}.Build(),
			}.Build(),
			want: &journal.Performance{Outflow: pcv{usd: -1.0}},
		},
		{
			desc: "inflow",
			trx: transaction.Builder{
				Postings: posting.Builder{
					Credit:    acc1,
					Debit:     portfolio,
					Value:     decimal.NewFromInt(1),
					Commodity: usd,
				}.Build(),
			}.Build(),
			want: &journal.Performance{Inflow: pcv{usd: 1.0}},
		},
		{
			desc: "dividend",
			trx: transaction.Builder{
				Targets: []*model.Commodity{aapl, usd},
				Postings: posting.Builder{
					Credit:    dividend,
					Debit:     portfolio,
					Value:     decimal.NewFromInt(1),
					Commodity: usd,
				}.Build(),
			}.Build(),
			want: &journal.Performance{
				InternalInflow:  pcv{usd: 1.0},
				InternalOutflow: pcv{aapl: -1.0},
			},
		},
		{
			desc: "expense",
			trx: transaction.Builder{
				Targets: []*model.Commodity{aapl, usd},
				Postings: posting.Builder{
					Credit:    portfolio,
					Debit:     expense,
					Value:     decimal.NewFromInt(1),
					Commodity: usd,
				}.Build(),
			}.Build(),
			want: &journal.Performance{
				InternalInflow:  pcv{aapl: 1.0},
				InternalOutflow: pcv{usd: -1.0},
			},
		},
		{
			desc: "expense with effect on porfolio",
			trx: transaction.Builder{
				Targets: make([]*model.Commodity, 0),
				Postings: posting.Builder{
					Credit:    portfolio,
					Debit:     expense,
					Value:     decimal.NewFromInt(1),
					Commodity: usd,
				}.Build(),
			}.Build(),
			want: &journal.Performance{
				InternalOutflow: pcv{usd: -1.0},
				PortfolioInflow: 1.0,
			},
		},
		{
			desc: "stock purchase",
			trx: transaction.Builder{

				Targets: []*model.Commodity{usd, aapl},
				Postings: posting.Builders{
					{
						Credit:    portfolio,
						Debit:     equity,
						Value:     decimal.NewFromInt(1010),
						Commodity: usd,
					},
					{
						Credit:    equity,
						Debit:     portfolio,
						Value:     decimal.NewFromInt(1000),
						Commodity: aapl,
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
			trx: transaction.Builder{
				Targets: []*model.Commodity{usd, aapl},
				Postings: posting.Builders{
					{
						Credit:    portfolio,
						Debit:     equity,
						Value:     decimal.NewFromInt(1010),
						Commodity: usd,
					},
					{
						Credit:    equity,
						Debit:     portfolio,
						Value:     decimal.NewFromInt(1000),
						Commodity: aapl,
					},
					{
						Credit:    portfolio,
						Debit:     equity,
						Value:     decimal.NewFromInt(10),
						Commodity: usd,
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
			trx: transaction.Builder{
				Targets: []*model.Commodity{usd, aapl},
				Postings: posting.Builders{
					{
						Credit:    portfolio,
						Debit:     equity,
						Value:     decimal.NewFromInt(1000),
						Commodity: aapl,
					},
					{
						Credit:    equity,
						Debit:     portfolio,
						Value:     decimal.NewFromInt(990),
						Commodity: usd,
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
			trx: transaction.Builder{
				Targets: []*model.Commodity{usd, gbp},
				Postings: posting.Builders{
					{
						Credit:    portfolio,
						Debit:     equity,
						Value:     decimal.NewFromInt(1400),
						Commodity: gbp,
					},
					{
						Credit:    equity,
						Debit:     portfolio,
						Value:     decimal.NewFromInt(1350),
						Commodity: usd,
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
			trx: transaction.Builder{
				Targets: []*model.Commodity{usd, gbp},
				Postings: posting.Builders{
					{
						Credit:    portfolio,
						Debit:     equity,
						Value:     decimal.NewFromInt(1400),
						Commodity: gbp,
					},
					{
						Credit:    equity,
						Debit:     portfolio,
						Value:     decimal.NewFromInt(1350),
						Commodity: usd,
					},
					{
						Credit:    portfolio,
						Debit:     expense,
						Value:     decimal.NewFromInt(10),
						Commodity: chf,
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
			trx: transaction.Builder{
				Targets: []*model.Commodity{usd, gbp},
				Postings: posting.Builders{
					{
						Credit:    portfolio,
						Debit:     equity,
						Value:     decimal.NewFromInt(1400),
						Commodity: gbp,
					},
					{
						Credit:    equity,
						Debit:     portfolio,
						Value:     decimal.NewFromInt(1350),
						Commodity: usd,
					},
					{
						Credit:    portfolio,
						Debit:     expense,
						Value:     decimal.NewFromInt(10),
						Commodity: usd,
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
				Transactions: []*model.Transaction{test.trx},
			}
			calc := Calculator{
				AccountFilter: predicate.ByName[*model.Account]([]*regexp.Regexp{
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
