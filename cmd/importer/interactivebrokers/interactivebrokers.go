// Copyright 2021 Silvio Böhler
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
	"regexp"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"

	"github.com/sboehler/knut/cmd/flags"
	"github.com/sboehler/knut/cmd/importer"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/printer"
)

// CreateCmd creates the command.
func CreateCmd() *cobra.Command {
	var r runner
	var cmd = &cobra.Command{
		Use:   "us.interactivebrokers",
		Short: "Import Interactive Brokers account reports",
		Long: `In the account manager web UI, go to "Reports" and download an "Activity" statement for the
		desired period (under "Default Statements"). Select CSV as the file format.`,

		Args: cobra.ExactValidArgs(1),
		RunE: r.run,
	}
	r.setupFlags(cmd)
	return cmd
}

func init() {
	importer.Register(CreateCmd)
}

type runner struct {
	accountFlag, dividendFlag, taxFlag, feeFlag, interestFlag, tradingFlag flags.AccountFlag
}

func (r *runner) setupFlags(c *cobra.Command) {
	c.Flags().VarP(&r.accountFlag, "account", "a", "account name")
	c.Flags().VarP(&r.interestFlag, "interest", "i", "account name of the interest expense account")
	c.Flags().VarP(&r.dividendFlag, "dividend", "d", "account name of the dividend account")
	c.Flags().VarP(&r.taxFlag, "tax", "w", "account name of the withholding tax account")
	c.Flags().VarP(&r.feeFlag, "fee", "f", "account name of the fee account")
	c.Flags().VarP(&r.tradingFlag, "trading", "t", "account name of the trading gain / loss account")
	c.MarkFlagRequired("account")
	c.MarkFlagRequired("interest")
	c.MarkFlagRequired("dividend")
	c.MarkFlagRequired("trading")
	c.MarkFlagRequired("tax")
	c.MarkFlagRequired("fee")
}

func (r *runner) run(cmd *cobra.Command, args []string) error {
	var (
		ctx = journal.NewContext()
		err error
	)
	f, err := flags.OpenFile(args[0])
	if err != nil {
		return err
	}
	var p = parser{
		reader:  csv.NewReader(f),
		builder: journal.New(ctx),
	}
	if p.account, err = r.accountFlag.Value(ctx); err != nil {
		return err
	}
	if p.interest, err = r.interestFlag.Value(ctx); err != nil {
		return err
	}
	if p.dividend, err = r.dividendFlag.Value(ctx); err != nil {
		return err
	}
	if p.tax, err = r.taxFlag.Value(ctx); err != nil {
		return err
	}
	if p.fee, err = r.feeFlag.Value(ctx); err != nil {
		return err
	}
	if p.trading, err = r.tradingFlag.Value(ctx); err != nil {
		return err
	}
	if err = p.parse(); err != nil {
		return err
	}
	out := bufio.NewWriter(cmd.OutOrStdout())
	defer out.Flush()
	_, err = printer.New().PrintLedger(out, p.builder.SortedDays())
	return err
}

type parser struct {
	reader           *csv.Reader
	builder          *journal.Journal
	baseCurrency     *journal.Commodity
	dateFrom, dateTo time.Time

	account, dividend, tax, fee, interest, trading *journal.Account
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
	if ok, err := p.parseBaseCurrency(l); ok || err != nil {
		return err
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

type accountInformationField int

const (
	aiAccountInformation accountInformationField = iota
	aiHeader
	aiFieldName
	aiFieldValue
)

func (p *parser) parseBaseCurrency(r []string) (bool, error) {
	if !(r[aiAccountInformation] == "Account Information" &&
		r[aiHeader] == "Data" &&
		r[aiFieldName] == "Base Currency") {
		return false, nil
	}
	var err error
	if p.baseCurrency, err = p.builder.Context.GetCommodity(r[aiFieldValue]); err != nil {
		return false, err
	}
	return true, nil
}

type statementField int

const (
	stfStatement statementField = iota
	stfHeader
	stfFieldName
	stfFieldValue
)

func (p *parser) parseDate(r []string) (bool, error) {
	if !(r[stfStatement] == "Statement" &&
		r[stfHeader] == "Data" && r[stfFieldName] == "Period") {
		return false, nil
	}
	var (
		dates            = strings.Split(r[stfFieldValue], " - ")
		dateFrom, dateTo time.Time
		err              error
	)
	if dateFrom, err = time.Parse("January 2, 2006", dates[0]); err != nil {
		return false, err
	}
	if dateTo, err = time.Parse("January 2, 2006", dates[1]); err != nil {
		return false, err
	}
	p.dateFrom, p.dateTo = dateFrom, dateTo
	return true, nil
}

type tradesField int

const (
	tfTrades tradesField = iota
	tfHeader
	tfDataDiscriminator
	tfAssetCategory
	tfCurrency
	tfSymbol
	tfDateTime
	tfQuantity
	tfTPrice
	tfCPrice
	tfProceeds
	tfCommFee
	tfBasis
	tfRealizedPL
	tfRealizedPLPct
	tfMTMPL
	tfCode
)

func (p *parser) parseTrade(r []string) (bool, error) {
	if !(r[tfTrades] == "Trades" &&
		r[tfHeader] == "Data" &&
		r[tfDataDiscriminator] == "Order" &&
		r[tfAssetCategory] == "Stocks") {
		return false, nil
	}
	var (
		currency, stock           *journal.Commodity
		date                      time.Time
		desc                      string
		qty, price, proceeds, fee decimal.Decimal
		err                       error
	)
	if currency, err = p.builder.Context.GetCommodity(r[tfCurrency]); err != nil {
		return false, err
	}
	if stock, err = p.builder.Context.GetCommodity(r[tfSymbol]); err != nil {
		return false, err
	}
	date, err = parseDateFromDateTime(r[tfDateTime])
	if err != nil {
		return false, err
	}
	if qty, err = parseRoundedDecimal(r[tfQuantity]); err != nil {
		return false, err
	}
	if price, err = parseDecimal(r[tfTPrice]); err != nil {
		return false, err
	}
	if proceeds, err = parseRoundedDecimal(r[tfProceeds]); err != nil {
		return false, err
	}
	if fee, err = decimal.NewFromString(r[tfCommFee]); err != nil {
		return false, err
	}
	if qty.IsPositive() {
		desc = fmt.Sprintf("Buy %s %s @ %s %s", qty, stock.Name(), price, currency.Name())
	} else {
		desc = fmt.Sprintf("Sell %s %s @ %s %s", qty, stock.Name(), price, currency.Name())
	}
	p.builder.AddTransaction(journal.TransactionBuilder{
		Date:        date,
		Description: desc,
		Postings: []*journal.Posting{
			journal.PostingWithTargets(p.trading, p.account, stock, qty, []*journal.Commodity{stock, currency}),
			journal.PostingWithTargets(p.trading, p.account, currency, proceeds, []*journal.Commodity{stock, currency}),
			journal.PostingWithTargets(p.fee, p.account, currency, fee, []*journal.Commodity{stock, currency}),
		},
	}.Build())
	return true, nil
}

func (p *parser) parseForex(r []string) (bool, error) {
	if !(r[tfTrades] == "Trades" &&
		r[tfHeader] == "Data" &&
		r[tfDataDiscriminator] == "Order" &&
		r[tfAssetCategory] == "Forex") {
		return false, nil
	}
	if p.baseCurrency == nil {
		return false, fmt.Errorf("base currency is not defined")
	}
	var (
		currency, stock           *journal.Commodity
		date                      time.Time
		desc                      string
		qty, price, proceeds, fee decimal.Decimal
		err                       error
	)
	if currency, err = p.builder.Context.GetCommodity(r[tfCurrency]); err != nil {
		return false, err
	}
	if stock, err = p.builder.Context.GetCommodity(strings.SplitN(r[tfSymbol], ".", 2)[0]); err != nil {
		return false, err
	}
	if date, err = parseDateFromDateTime(r[tfDateTime]); err != nil {
		return false, err
	}
	if qty, err = parseRoundedDecimal(r[tfQuantity]); err != nil {
		return false, err
	}
	if price, err = parseDecimal(r[tfTPrice]); err != nil {
		return false, err
	}
	if proceeds, err = parseRoundedDecimal(r[tfProceeds]); err != nil {
		return false, err
	}
	if fee, err = parseRoundedDecimal(r[tfCommFee]); err != nil {
		return false, err
	}
	if qty.IsPositive() {
		desc = fmt.Sprintf("Buy %s %s @ %s %s", qty, stock.Name(), price, currency.Name())
	} else {
		desc = fmt.Sprintf("Sell %s %s @ %s %s", qty, stock.Name(), price, currency.Name())
	}
	var postings = []*journal.Posting{
		journal.PostingWithTargets(p.trading, p.account, stock, qty, []*journal.Commodity{stock, currency}),
		journal.PostingWithTargets(p.trading, p.account, currency, proceeds, []*journal.Commodity{stock, currency}),
	}
	if !fee.IsZero() {
		postings = append(postings, journal.PostingWithTargets(p.fee, p.account, p.baseCurrency, fee, []*journal.Commodity{stock, currency}))
	}
	p.builder.AddTransaction(journal.TransactionBuilder{
		Date:        date,
		Description: desc,
		Postings:    postings,
	}.Build())
	return true, nil
}

type depositsWithdrawalsField int

const (
	dwfDepositsWithdrawals depositsWithdrawalsField = iota
	dwfHeader
	dwfCurrency
	dwfSettleDate
	dwfDescription
	dwfAmount
)

func (p *parser) parseDepositOrWithdrawal(r []string) (bool, error) {
	if !(r[dwfDepositsWithdrawals] == "Deposits & Withdrawals" &&
		r[dwfHeader] == "Data" &&
		r[dwfCurrency] != "Total" &&
		r[dwfSettleDate] != "") {
		return false, nil
	}
	var (
		currency *journal.Commodity
		date     time.Time
		desc     string
		amount   decimal.Decimal
		err      error
	)
	if currency, err = p.builder.Context.GetCommodity(r[dwfCurrency]); err != nil {
		return false, err
	}
	if date, err = parseDate(r[dwfSettleDate]); err != nil {
		return false, err
	}
	if amount, err = parseRoundedDecimal(r[dwfAmount]); err != nil {
		return false, err
	}
	if amount.IsPositive() {
		desc = fmt.Sprintf("Deposit %s %s", amount, currency.Name())
	} else {
		desc = fmt.Sprintf("Withdraw %s %s", amount, currency.Name())
	}
	p.builder.AddTransaction(journal.TransactionBuilder{
		Date:        date,
		Description: desc,
		Postings: []*journal.Posting{
			journal.NewPosting(p.builder.Context.TBDAccount(), p.account, currency, amount),
		},
	}.Build())
	return true, nil
}

type dividendsField int

const (
	dfDividends dividendsField = iota
	dfHeader
	dfCurrency
	dfDate
	dfDescription
	dfAmount
)

func (p *parser) parseDividend(r []string) (bool, error) {
	if !(r[dfDividends] == "Dividends" &&
		r[dfHeader] == "Data" &&
		!strings.HasPrefix(r[dfCurrency], "Total") &&
		len(r) == 6) {
		return false, nil
	}
	var (
		currency, security *journal.Commodity
		date               time.Time
		desc               = r[dfDescription]
		amount             decimal.Decimal
		symbol             string
		err                error
	)
	if currency, err = p.builder.Context.GetCommodity(r[dfCurrency]); err != nil {
		return false, err
	}
	if date, err = parseDate(r[dfDate]); err != nil {
		return false, err
	}
	if amount, err = parseDecimal(r[dfAmount]); err != nil {
		return false, err
	}
	if symbol, err = parseDividendSymbol(r[dfDescription]); err != nil {
		return false, err
	}
	if security, err = p.builder.Context.GetCommodity(symbol); err != nil {
		return false, err
	}
	p.builder.AddTransaction(journal.TransactionBuilder{
		Date:        date,
		Description: desc,
		Postings: []*journal.Posting{
			journal.PostingWithTargets(p.dividend, p.account, currency, amount, []*journal.Commodity{security}),
		},
	}.Build())
	return true, nil
}

var dividendSymbolRegex = regexp.MustCompile("[A-Za-z0-9]+")

func parseDividendSymbol(s string) (string, error) {
	var symbol = dividendSymbolRegex.FindString(s)
	if symbol == "" {
		return symbol, fmt.Errorf("invalid symbol name %s", s)
	}
	return symbol, nil
}

type withholdingTaxField int

const (
	wtfWithholdingTax withholdingTaxField = iota
	wtfHeader
	wtfCurrency
	wtfDate
	wtfDescription
	wtfAmount
	wtfCode
)

func (p *parser) parseWithholdingTax(r []string) (bool, error) {
	if !(r[wtfWithholdingTax] == "Withholding Tax" &&
		r[wtfHeader] == "Data" &&
		!strings.HasPrefix(r[wtfCurrency], "Total")) {
		return false, nil
	}
	var (
		desc               = r[wtfDescription]
		currency, security *journal.Commodity
		date               time.Time
		amount             decimal.Decimal
		symbol             string
		err                error
	)
	if currency, err = p.builder.Context.GetCommodity(r[wtfCurrency]); err != nil {
		return false, err
	}
	if date, err = parseDate(r[wtfDate]); err != nil {
		return false, err
	}
	if amount, err = parseDecimal(r[wtfAmount]); err != nil {
		return false, err
	}
	if symbol, err = parseDividendSymbol(r[wtfDescription]); err != nil {
		return false, err
	}
	if security, err = p.builder.Context.GetCommodity(symbol); err != nil {
		return false, err
	}
	p.builder.AddTransaction(journal.TransactionBuilder{
		Date:        date,
		Description: desc,
		Postings: []*journal.Posting{
			journal.PostingWithTargets(p.tax, p.account, currency, amount, []*journal.Commodity{security}),
		},
	}.Build())
	return true, nil
}

// Interest,Data,USD,2020-07-06,USD Debit Interest for Jun-2020,-0.73
func (p *parser) parseInterest(r []string) (bool, error) {
	if !(r[dfDividends] == "Interest" && r[dfHeader] == "Data" && !strings.HasPrefix(r[dfCurrency], "Total") && len(r) == 6) {
		return false, nil
	}
	var (
		currency *journal.Commodity
		date     time.Time
		amount   decimal.Decimal
		desc     = r[dfDescription]
		err      error
	)
	if currency, err = p.builder.Context.GetCommodity(r[dfCurrency]); err != nil {
		return false, err
	}
	if date, err = parseDate(r[dfDate]); err != nil {
		return false, err
	}
	if amount, err = parseDecimal(r[dfAmount]); err != nil {
		return false, err
	}
	p.builder.AddTransaction(journal.TransactionBuilder{
		Date:        date,
		Description: desc,
		Postings: []*journal.Posting{
			journal.PostingWithTargets(p.interest, p.account, currency, amount, []*journal.Commodity{currency})},
	}.Build())
	return true, nil
}

type openPositionsField int

const (
	opfOpenPositions openPositionsField = iota
	opfHeader
	opfDataDiscriminator
	opfAssetCategory
	opfCurrency
	opfSymbol
	opfQuantity
	opfMult
	opfCostPrice
	opfCostBasis
	opfClosePrice
	opfValue
	opfUnrealizedPL
	opfUnrealizedPLPct
	opfCode
)

func (p *parser) createAssertions(r []string) (bool, error) {
	if !(r[opfOpenPositions] == "Open Positions" &&
		r[opfHeader] == "Data" &&
		r[opfDataDiscriminator] == "Summary") {
		return false, nil
	}
	if p.dateTo.IsZero() {
		return false, fmt.Errorf("report end date has not been parsed yet")
	}
	var (
		symbol *journal.Commodity
		amt    decimal.Decimal
		err    error
	)
	if symbol, err = p.builder.Context.GetCommodity(r[opfSymbol]); err != nil {
		return false, err
	}
	if amt, err = decimal.NewFromString(r[opfQuantity]); err != nil {
		return false, err
	}
	p.builder.AddAssertion(&journal.Assertion{
		Date:      p.dateTo,
		Account:   p.account,
		Commodity: symbol,
		Amount:    amt,
	})
	return true, nil
}

type forexBalancesField int

const (
	fbfForexBalances forexBalancesField = iota
	fbfHeader
	fbfAssetCategory
	fbfCurrency
	fbfDescription
	fbfQuantity
	fbfCostPrice
	fbfCostBasisInCHF
	fbfClosePrice
	fbfValueInCHF
	fbfUnrealizedPLInCHF
	fbfCode
)

func (p *parser) createCurrencyAssertions(r []string) (bool, error) {
	if !(r[fbfForexBalances] == "Forex Balances" &&
		r[fbfHeader] == "Data" &&
		r[fbfAssetCategory] == "Forex") {
		return false, nil
	}
	if p.dateTo.IsZero() {
		return false, fmt.Errorf("report end date has not been parsed yet")
	}
	var (
		symbol *journal.Commodity
		amount decimal.Decimal
		err    error
	)
	if symbol, err = p.builder.Context.GetCommodity(r[fbfDescription]); err != nil {
		return false, err
	}
	if amount, err = parseRoundedDecimal(r[fbfQuantity]); err != nil {
		return false, err
	}
	p.builder.AddAssertion(&journal.Assertion{
		Date:      p.dateTo,
		Account:   p.account,
		Commodity: symbol,
		Amount:    amount,
	})
	return true, nil
}

func parseRoundedDecimal(s string) (decimal.Decimal, error) {
	amount, err := parseDecimal(s)
	if err != nil {
		return amount, err
	}
	return amount.Round(2), nil
}

func parseDecimal(s string) (decimal.Decimal, error) {
	return decimal.NewFromString(strings.ReplaceAll(s, ",", ""))
}

func parseDateFromDateTime(s string) (time.Time, error) {
	return parseDate(s[:10])
}

func parseDate(s string) (time.Time, error) {
	return time.Parse("2006-01-02", s)
}
