package demo

import (
	"bytes"
	"fmt"
	"math"
	"math/rand"
	"os"
	"time"

	"github.com/natefinch/atomic"
	"github.com/sboehler/knut/cmd/flags"
	"github.com/sboehler/knut/lib/common/date"
	"github.com/sboehler/knut/lib/journal"
	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"
)

// CreateCmd creates the command.
func CreateCmd() *cobra.Command {

	var r runner

	// Cmd is the balance command.
	c := &cobra.Command{
		Use:    "demo",
		Short:  "create a demo journal",
		Long:   `Create an example journal for knut.`,
		Args:   cobra.ExactValidArgs(1),
		Run:    r.run,
		Hidden: true,
	}
	r.setupFlags(c)
	return c
}

type runner struct {
	period flags.PeriodFlag
	seed   int64
}

func (r *runner) run(cmd *cobra.Command, args []string) {
	if err := r.execute(cmd, args); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "%+v\n", err)
		os.Exit(1)
	}
}

func (r *runner) setupFlags(c *cobra.Command) {
	r.period.Setup(c, date.Period{Start: date.Date(2018, 1, 1), End: date.Date(2021, 12, 31)})
	c.Flags().Int64Var(&r.seed, "seed", 0, "random seed to create reproducible results")
}

const (
	accSalary      = "Income:Salary"
	accDividends   = "Income:Investents:Dividends"
	accInterest    = "Income:Investments:Interest"
	accBankAccount = "Assets:BankAccount"
	accPortfolio   = "Assets:Portfolio"
	accRent        = "Expenses:Rent"
	accGroceries   = "Expenses:Groceries"
	accTaxes       = "Expenses:Taxes"
	accEquity      = "Equity:Equity"
)

const (
	comUSD    = "USD"
	comKNUTCO = "KNUTCO"
)

var (
	accountNames = []string{
		accSalary,
		accDividends,
		accInterest,
		accBankAccount,
		accPortfolio,
		accRent,
		accGroceries,
		accTaxes,
		accEquity,
	}
	commodityNames = []string{
		comUSD, comKNUTCO,
	}
)

func (r *runner) execute(cmd *cobra.Command, args []string) error {
	if r.seed == 0 {
		r.seed = time.Now().UnixNano()
	}
	period := r.period.Value()
	monthEndDates := date.NewPartition(period, date.Monthly, 0).EndDates()
	prevDate := period.Start.AddDate(0, 0, -1)
	jctx := journal.NewContext()
	j := journal.New(jctx)

	// open accounts
	accounts := make(map[string]*journal.Account)
	for _, acc := range accountNames {
		accounts[acc] = jctx.Account(string(acc))
		j.AddOpen(&journal.Open{Date: prevDate, Account: accounts[acc]})
	}

	// create commodities
	commodities := make(map[string]*journal.Commodity)
	for _, com := range commodityNames {
		commodities[com] = jctx.Commodity(string(com))
	}

	// initialize opening balances
	j.AddTransaction(journal.TransactionBuilder{
		Date:        prevDate,
		Description: fmt.Sprintf("Opening balance for %s", accBankAccount),
		Postings: journal.PostingBuilder{
			Credit:    jctx.Account("Equity:Equity"),
			Debit:     accounts[accBankAccount],
			Amount:    decimal.NewFromInt(12500),
			Commodity: commodities[comUSD],
		}.Build(),
	}.Build())
	j.AddTransaction(journal.TransactionBuilder{
		Date:        prevDate,
		Description: fmt.Sprintf("Opening balance for %s", accPortfolio),
		Postings: journal.PostingBuilder{
			Credit:    jctx.Account("Equity:Equity"),
			Debit:     accounts[accPortfolio],
			Amount:    decimal.NewFromInt(100),
			Commodity: commodities[comKNUTCO],
		}.Build(),
	}.Build())

	rnd := rand.New(rand.NewSource(r.seed))
	price := (1 + rnd.Float64()) * 100
	j.AddPrice(&journal.Price{
		Date:      prevDate,
		Commodity: commodities[comKNUTCO],
		Price:     decimal.NewFromFloat(price),
		Target:    commodities[comUSD],
	})
	for _, eom := range monthEndDates {
		// pay salary on last day of month
		j.AddTransaction(journal.TransactionBuilder{
			Date:        eom,
			Description: "Salary",
			Postings: journal.PostingBuilder{
				Credit:    accounts[accSalary],
				Debit:     accounts[accBankAccount],
				Amount:    decimal.NewFromInt(4350),
				Commodity: commodities[comUSD],
			}.Build(),
		}.Build())

		som := date.StartOf(eom, date.Monthly)

		// pay rent on first day of month
		j.AddTransaction(journal.TransactionBuilder{
			Date:        som,
			Description: "Rent",
			Postings: journal.PostingBuilder{
				Credit:    accounts[accBankAccount],
				Debit:     accounts[accRent],
				Amount:    decimal.NewFromInt(980),
				Commodity: commodities[comUSD],
			}.Build(),
		}.Build())

		if eom.Month() == time.December {
			acc := jctx.Account(fmt.Sprintf("Liabilities:Taxes:Y%d", eom.Year()))
			j.AddOpen(&journal.Open{
				Date:    date.Date(eom.Year(), 1, 1),
				Account: acc,
			})
			// pay annual tax in december, accrued over all months
			t := journal.TransactionBuilder{
				Date:        eom,
				Description: "Taxes",
				Postings: journal.PostingBuilder{
					Credit:    accounts[accBankAccount],
					Debit:     accounts[accTaxes],
					Amount:    decimal.NewFromInt(12000),
					Commodity: commodities[comUSD],
				}.Build(),
			}.Build()
			t.Accrual = &journal.Accrual{
				Interval: date.Monthly,
				Period:   date.Period{Start: date.Date(eom.Year(), 1, 1), End: date.Date(eom.Year(), 12, 31)},
				Account:  acc,
			}
			j.AddTransaction(t)
		}

		ds := date.NewPartition(date.Period{Start: som, End: eom}, date.Daily, 0).EndDates()
		for _, d := range ds {

			if rnd.Intn(100) < 20 {
				// go shopping
				j.AddTransaction(journal.TransactionBuilder{
					Date:        d,
					Description: "Shopping",
					Postings: journal.PostingBuilder{
						Credit:    accounts[accBankAccount],
						Debit:     accounts[accGroceries],
						Amount:    decimal.NewFromInt(int64(rnd.Intn(200))),
						Commodity: commodities[comUSD],
					}.Build(),
				}.Build())
			}

			// update prices
			price = price * (1 + rnd.NormFloat64()*0.23/math.Sqrt(365))
			j.AddPrice(&journal.Price{
				Date:      d,
				Commodity: commodities[comKNUTCO],
				Price:     decimal.NewFromFloat(price),
				Target:    commodities[comUSD],
			})
		}
	}

	p := journal.NewPrinter()
	var buf bytes.Buffer
	p.PrintJournal(&buf, j)
	return atomic.WriteFile(args[0], &buf)
}
