// Copyright 2020 Silvio Böhler
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package interactivebrokers

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"

	"github.com/sboehler/knut/cmd/importer"
	"github.com/sboehler/knut/lib/ledger"
	"github.com/sboehler/knut/lib/model/accounts"
	"github.com/sboehler/knut/lib/model/commodities"
	"github.com/sboehler/knut/lib/printer"
	"github.com/sboehler/knut/lib/scanner"
)

// CreateCmd creates the command.
func CreateCmd() *cobra.Command {
	var cmd = cobra.Command{
		Use:   "us.interactivebrokers",
		Short: "Import Interactive Brokers account reports",
		Long: `In the account manager web UI, go to "Reports" and download an "Activity" statement for the
		desired period (under "Default Statements"). Select CSV as the file format.`,

		Args: cobra.ExactValidArgs(1),
		RunE: run,
	}
	cmd.Flags().StringP("account", "a", "", "account name")
	cmd.Flags().StringP("interest", "i", "Expenses:TBD", "account name of the interest expense account")
	cmd.Flags().StringP("dividend", "d", "Expenses:TBD", "account name of the dividend account")
	cmd.Flags().StringP("tax", "t", "Expenses:TBD", "account name of the withholding tax account")
	cmd.Flags().StringP("fee", "f", "Expenses:TBD", "account name of the fee account")
	return &cmd
}

func init() {
	importer.Register(CreateCmd)
}

type options struct {
	account, dividend, tax, fee, interest *accounts.Account
}

func run(cmd *cobra.Command, args []string) error {
	var (
		o   options
		err error
	)
	if o.account, err = getAccountFlag(cmd, "account"); err != nil {
		return err
	}
	if o.dividend, err = getAccountFlag(cmd, "dividend"); err != nil {
		return err
	}
	if o.interest, err = getAccountFlag(cmd, "interest"); err != nil {
		return err
	}
	if o.tax, err = getAccountFlag(cmd, "tax"); err != nil {
		return err
	}
	if o.fee, err = getAccountFlag(cmd, "fee"); err != nil {
		return err
	}
	f, err := os.Open(args[0])
	if err != nil {
		return err
	}
	var (
		reader = csv.NewReader(bufio.NewReader(f))
		p      = parser{
			reader:  reader,
			options: o,
			builder: ledger.NewBuilder(ledger.Filter{}),
		}
	)
	if err = p.parse(); err != nil {
		return err
	}
	var w = bufio.NewWriter(cmd.OutOrStdout())
	defer w.Flush()
	_, err = printer.PrintLedger(w, p.builder.Build())
	return err
}

func getAccountFlag(cmd *cobra.Command, flag string) (*accounts.Account, error) {
	name, err := cmd.Flags().GetString(flag)
	if err != nil {
		return nil, err
	}
	s, err := scanner.New(strings.NewReader(name), "")
	if err != nil {
		return nil, err
	}
	return scanner.ParseAccount(s)
}

type parser struct {
	reader           *csv.Reader
	options          options
	builder          *ledger.Builder
	baseCurrency     *commodities.Commodity
	dateFrom, dateTo time.Time
}

func (p *parser) parse() error {
	// variable number of fields per line
	p.reader.FieldsPerRecord = -1
	// quotes can appear within fields
	p.reader.LazyQuotes = true
	for {
		err := p.readLine()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

func (p *parser) readLine() error {
	l, err := p.reader.Read()
	if err != nil {
		return err
	}
	if ok := p.parseBaseCurrency(l); ok {
		return nil
	}
	if ok, err := p.parseDate(l); ok || err != nil {
		return err
	}
	if ok, err := p.parseForex(l); ok || err != nil {
		return err
	}
	if ok, err := p.parseTrade(l); ok || err != nil {
		return err
	}
	if ok, err := p.parseDepositOrWithdrawal(l); ok || err != nil {
		return err
	}
	if ok, err := p.parseDividend(l); ok || err != nil {
		return err
	}
	if ok, err := p.parseInterest(l); ok || err != nil {
		return err
	}
	if ok, err := p.parseWithholdingTax(l); ok || err != nil {
		return err
	}
	if ok, err := p.createAssertions(l); ok || err != nil {
		return err
	}
	if ok, err := p.createCurrencyAssertions(l); ok || err != nil {
		return err
	}
	return nil
}

func (p *parser) parseBaseCurrency(r []string) bool {
	if r[0] != "Account Information" || r[1] != "Data" || r[2] != "Base Currency" {
		return false
	}
	p.baseCurrency = commodities.Get(r[3])
	return true
}

func (p *parser) parseDate(r []string) (bool, error) {
	if r[0] != "Statement" || r[1] != "Data" || r[2] != "Period" {
		return false, nil
	}
	ds := strings.Split(r[3], " - ")
	var err error
	p.dateFrom, err = time.Parse("January 2, 2006", ds[0])
	if err != nil {
		return false, err
	}
	p.dateTo, err = time.Parse("January 2, 2006", ds[1])
	if err != nil {
		return false, err
	}
	return true, nil
}

func (p *parser) parseTrade(r []string) (bool, error) {
	if r[0] != "Trades" || r[1] != "Data" || r[2] != "Order" || r[3] != "Stocks" {
		return false, nil
	}
	var (
		currency = commodities.Get(r[4])
		stock    = commodities.Get(r[5])
	)
	date, err := time.Parse("2006-01-02", r[6][:10])
	if err != nil {
		return false, err
	}
	qty, err := decimal.NewFromString(r[7])
	if err != nil {
		return false, err
	}
	price, err := decimal.NewFromString(r[8])
	if err != nil {
		return false, err
	}
	proceeds, err := decimal.NewFromString(strings.ReplaceAll(r[10], ",", ""))
	if err != nil {
		return false, err
	}
	fee, err := decimal.NewFromString(r[11])
	if err != nil {
		return false, err
	}
	var desc string
	if qty.IsPositive() {
		desc = fmt.Sprintf("Buy %s %s @ %s %s", qty, stock, price, currency)
	} else {
		desc = fmt.Sprintf("Sell %s %s @ %s %s", qty, stock, price, currency)
	}
	p.builder.AddTransaction(&ledger.Transaction{
		Date:        date,
		Description: desc,
		Postings: []*ledger.Posting{
			ledger.NewPosting(accounts.EquityAccount(), p.options.account, stock, qty.Round(2)),
			ledger.NewPosting(accounts.EquityAccount(), p.options.account, currency, proceeds.Round(2)),
			ledger.NewPosting(p.options.fee, p.options.account, currency, fee.Round(2)),
		},
	})
	return true, nil
}

func (p *parser) parseForex(r []string) (bool, error) {
	if r[0] != "Trades" || r[1] != "Data" || r[2] != "Order" || r[3] != "Forex" {
		return false, nil
	}
	if p.baseCurrency == nil {
		return false, fmt.Errorf("base currency is not defined")
	}
	var (
		currency  = commodities.Get(r[4])
		currency2 = strings.SplitN(r[5], ".", 2)[0]
		stock     = commodities.Get(currency2)
	)
	date, err := time.Parse("2006-01-02", r[6][:10])
	if err != nil {
		return false, err
	}
	qty, err := decimal.NewFromString(strings.ReplaceAll(r[7], ",", ""))
	if err != nil {
		return false, err
	}
	price, err := decimal.NewFromString(r[8])
	if err != nil {
		return false, err
	}
	proceeds, err := decimal.NewFromString(strings.ReplaceAll(r[10], ",", ""))
	if err != nil {
		return false, err
	}
	fee, err := decimal.NewFromString(r[11])
	if err != nil {
		return false, err
	}
	var desc string
	if qty.IsPositive() {
		desc = fmt.Sprintf("Buy %s %s @ %s %s", qty, stock, price, currency)
	} else {
		desc = fmt.Sprintf("Sell %s %s @ %s %s", qty, stock, price, currency)
	}
	var postings = []*ledger.Posting{
		ledger.NewPosting(accounts.EquityAccount(), p.options.account, stock, qty.Round(2)),
		ledger.NewPosting(accounts.EquityAccount(), p.options.account, currency, proceeds.Round(2)),
	}
	if !fee.IsZero() {
		postings = append(postings, ledger.NewPosting(p.options.fee, p.options.account, p.baseCurrency, fee.Round(2)))
	}
	p.builder.AddTransaction(&ledger.Transaction{
		Date:        date,
		Description: desc,
		Postings:    postings,
	})
	return true, nil
}

func (p *parser) parseDepositOrWithdrawal(r []string) (bool, error) {
	if r[0] != "Deposits & Withdrawals" || r[1] != "Data" || r[2] == "Total" || r[3] == "" {
		return false, nil
	}
	var currency = commodities.Get(r[2])
	date, err := time.Parse("2006-01-02", r[3])
	if err != nil {
		return false, err
	}
	amt, err := decimal.NewFromString(r[5])
	if err != nil {
		return false, err
	}
	var desc string
	if amt.IsPositive() {
		desc = fmt.Sprintf("Deposit %s %s", amt, currency)
	} else {
		desc = fmt.Sprintf("Withdraw %s %s", amt, currency)
	}
	p.builder.AddTransaction(&ledger.Transaction{
		Date:        date,
		Description: desc,
		Postings: []*ledger.Posting{
			ledger.NewPosting(accounts.TBDAccount(), p.options.account, currency, amt),
		},
	})
	return true, nil
}

func (p *parser) parseDividend(r []string) (bool, error) {
	if r[0] != "Dividends" || r[1] != "Data" || strings.HasPrefix(r[2], "Total") || len(r) != 6 {
		return false, nil
	}
	var currency = commodities.Get(r[2])
	date, err := time.Parse("2006-01-02", r[3])
	if err != nil {
		return false, err
	}
	amt, err := decimal.NewFromString(r[5])
	if err != nil {
		return false, err
	}
	var (
		regex  = regexp.MustCompile("[A-Za-z0-9]+")
		symbol = strings.TrimSpace(strings.Split(r[4], "(")[0])
	)
	if !regex.MatchString(symbol) {
		return false, fmt.Errorf("invalid symbol name %s", symbol)
	}
	var (
		security = commodities.Get(symbol)
		desc     = r[4]
	)
	p.builder.AddTransaction(&ledger.Transaction{
		Date:        date,
		Description: desc,
		Postings: []*ledger.Posting{
			ledger.NewPosting(p.options.dividend, p.options.account, currency, amt),
			ledger.NewPosting(p.options.dividend, p.options.dividend, security, decimal.Zero),
		},
	})
	return true, nil
}

func (p *parser) parseWithholdingTax(r []string) (bool, error) {
	if r[0] != "Withholding Tax" || r[1] != "Data" || strings.HasPrefix(r[2], "Total") {
		return false, nil
	}
	var currency = commodities.Get(r[2])
	date, err := time.Parse("2006-01-02", r[3])
	if err != nil {
		return false, err
	}
	amt, err := decimal.NewFromString(r[5])
	if err != nil {
		return false, err
	}
	var (
		regex  = regexp.MustCompile("[A-Za-z0-9]+")
		symbol = strings.TrimSpace(strings.Split(r[4], "(")[0])
	)
	if !regex.MatchString(symbol) {
		return false, fmt.Errorf("invalid symbol name %s", symbol)
	}
	var (
		security = commodities.Get(symbol)
		desc     = r[4]
	)
	p.builder.AddTransaction(&ledger.Transaction{
		Date:        date,
		Description: desc,
		Postings: []*ledger.Posting{
			ledger.NewPosting(p.options.tax, p.options.account, currency, amt),
			ledger.NewPosting(p.options.tax, p.options.tax, security, decimal.Zero),
		},
	})
	return true, nil
}

//Interest,Data,USD,2020-07-06,USD Debit Interest for Jun-2020,-0.73
func (p *parser) parseInterest(r []string) (bool, error) {
	if r[0] != "Interest" || r[1] != "Data" || strings.HasPrefix(r[2], "Total") || len(r) != 6 {
		return false, nil
	}
	var currency = commodities.Get(r[2])
	date, err := time.Parse("2006-01-02", r[3])
	if err != nil {
		return false, err
	}
	amt, err := decimal.NewFromString(r[5])
	if err != nil {
		return false, err
	}
	var desc = r[4]
	p.builder.AddTransaction(&ledger.Transaction{
		Date:        date,
		Description: desc,
		Postings: []*ledger.Posting{
			ledger.NewPosting(p.options.interest, p.options.account, currency, amt)},
	})
	return true, nil
}

func (p *parser) createAssertions(r []string) (bool, error) {
	if r[0] != "Open Positions" || r[1] != "Data" || r[2] != "Summary" {
		return false, nil
	}
	if p.dateTo.IsZero() {
		return false, fmt.Errorf("report end date has not been parsed yet")
	}
	var symbol = commodities.Get(r[5])
	amt, err := decimal.NewFromString(r[6])
	if err != nil {
		return false, err
	}
	p.builder.AddAssertion(&ledger.Assertion{
		Date:      p.dateTo,
		Account:   p.options.account,
		Commodity: symbol,
		Amount:    amt,
	})
	return true, nil
}

func (p *parser) createCurrencyAssertions(r []string) (bool, error) {
	if r[0] != "Forex Balances" || r[1] != "Data" || r[2] != "Forex" {
		return false, nil
	}
	if p.dateTo.IsZero() {
		return false, fmt.Errorf("report end date has not been parsed yet")
	}
	symbol := commodities.Get(r[4])
	amt, err := decimal.NewFromString(r[5])
	if err != nil {
		return false, err
	}
	p.builder.AddAssertion(&ledger.Assertion{
		Date:      p.dateTo,
		Account:   p.options.account,
		Commodity: symbol,
		Amount:    amt.Round(2),
	})
	return true, nil
}
