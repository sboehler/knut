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
	"github.com/sboehler/knut/lib/model"
	"github.com/sboehler/knut/lib/model/posting"
	"github.com/sboehler/knut/lib/model/registry"
	"github.com/sboehler/knut/lib/model/transaction"
)

// CreateCmd creates the command.
func CreateCmd() *cobra.Command {
	var r runner
	cmd := &cobra.Command{
		Use:   "us.interactivebrokers",
		Short: "Import Interactive Brokers account reports",
		Long: `In the account manager web UI, go to "Reports" and download an "Activity" statement for the
		desired period (under "Default Statements"). Select CSV as the file format.`,

		Args: cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: r.run,
	}
	r.setupFlags(cmd)
	return cmd
}

func init() {
	importer.RegisterImporter(CreateCmd)
}

type runner struct {
	accountFlag, dividendFlag, taxFlag, feeFlag, interestFlag, tradingFlag, transferFlag flags.AccountFlag
}

func (r *runner) setupFlags(c *cobra.Command) {
	c.Flags().VarP(&r.accountFlag, "account", "a", "account name")
	c.Flags().VarP(&r.interestFlag, "interest", "i", "account name of the interest expense account")
	c.Flags().VarP(&r.dividendFlag, "dividend", "d", "account name of the dividend account")
	c.Flags().VarP(&r.taxFlag, "tax", "w", "account name of the withholding tax account")
	c.Flags().VarP(&r.feeFlag, "fee", "f", "account name of the fee account")
	c.Flags().VarP(&r.tradingFlag, "trading", "t", "account name of the trading gain / loss account")
	c.Flags().VarP(&r.transferFlag, "transfer", "x", "account name of the transfer account")
	c.MarkFlagRequired("account")
	c.MarkFlagRequired("interest")
	c.MarkFlagRequired("dividend")
	c.MarkFlagRequired("trading")
	c.MarkFlagRequired("tax")
	c.MarkFlagRequired("fee")
}

func (r *runner) run(cmd *cobra.Command, args []string) error {
	var (
		reg = registry.New()
		err error
	)
	f, err := flags.OpenFile(args[0])
	if err != nil {
		return err
	}
	p := parser{
		registry: reg,
		reader:   csv.NewReader(f),
		builder:  journal.New(),
	}
	if p.account, err = r.accountFlag.Value(reg.Accounts()); err != nil {
		return err
	}
	if p.interest, err = r.interestFlag.Value(reg.Accounts()); err != nil {
		return err
	}
	if p.dividend, err = r.dividendFlag.Value(reg.Accounts()); err != nil {
		return err
	}
	if p.tax, err = r.taxFlag.Value(reg.Accounts()); err != nil {
		return err
	}
	if p.fee, err = r.feeFlag.Value(reg.Accounts()); err != nil {
		return err
	}
	if p.trading, err = r.tradingFlag.Value(reg.Accounts()); err != nil {
		return err
	}
	if p.transfer, err = r.transferFlag.Value(reg.Accounts()); err != nil {
		return err
	}
	if p.transfer == nil {
		p.transfer = p.registry.Accounts().TBDAccount()
	}
	if err = p.parse(); err != nil {
		return err
	}
	out := bufio.NewWriter(cmd.OutOrStdout())
	defer out.Flush()
	return journal.Print(out, p.builder.Build())
}

type parser struct {
	registry         *model.Registry
	reader           *csv.Reader
	builder          *journal.Builder
	baseCurrency     *model.Commodity
	dateFrom, dateTo time.Time
	hdr              []string

	account, dividend, tax, fee, interest, trading, transfer *model.Account
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

const (
	lData    = "Data"
	lSection = "Section"
	lHeader  = "Header"
)

func (p *parser) readLine() error {
	l, err := p.reader.Read()
	if err != nil {
		return err
	}
	// remove byte order mark if present
	ByteOrderMarkAsString := string('\uFEFF')
	l[0] = strings.TrimPrefix(l[0], ByteOrderMarkAsString)

	switch l[1] {
	case lHeader:
		p.hdr = l
		return nil
	case lData:
		if l[0] != p.hdr[0] {
			// ignoring invalid data
			return nil
		}
	default:
		return nil
	}

	fields := make(map[string]string)

	for k, v := range p.hdr {
		if v == "" {
			continue
		}
		if k >= len(l) {
			continue
		}
		if k == 0 {
			fields[lSection] = l[k]
			continue
		}
		if k == 1 {
			fields[lHeader] = l[k]
			continue
		}
		fields[v] = l[k]
	}
	if ok, err := p.parseBaseCurrency(fields); ok || err != nil {
		return err
	}
	if ok, err := p.parseDate(fields); ok || err != nil {
		return err
	}
	if ok, err := p.parseForex(fields); ok || err != nil {
		return err
	}
	if ok, err := p.parseTrade(fields); ok || err != nil {
		return err
	}
	if ok, err := p.parseDepositOrWithdrawal(fields); ok || err != nil {
		return err
	}
	if ok, err := p.parseDividend(fields); ok || err != nil {
		return err
	}
	if ok, err := p.parseInterest(fields); ok || err != nil {
		return err
	}
	if ok, err := p.parseWithholdingTax(fields); ok || err != nil {
		return err
	}
	if ok, err := p.parseFees(fields); ok || err != nil {
		return err
	}
	if ok, err := p.createAssertions(fields); ok || err != nil {
		return err
	}
	if ok, err := p.createCurrencyAssertions(fields); ok || err != nil {
		return err
	}
	return nil
}

const (
	aiFieldName  = "Field Name"
	aiFieldValue = "Field Value"
)

func (p *parser) parseBaseCurrency(r map[string]string) (bool, error) {
	if !(r[lSection] == "Account Information" &&
		r[lHeader] == "Data" &&
		r[aiFieldName] == "Base Currency") {
		return false, nil
	}
	var err error
	if p.baseCurrency, err = p.registry.Commodities().Get(r[aiFieldValue]); err != nil {
		return false, err
	}
	return true, nil
}

const (
	stfFieldName  = "Field Name"
	stfFieldValue = "Field Value"
)

func (p *parser) parseDate(r map[string]string) (bool, error) {
	if !(r[lSection] == "Statement" &&
		r[lHeader] == "Data" && r[stfFieldName] == "Period") {
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

const (
	tfDataDiscriminator = "DataDiscriminator"
	tfAssetCategory     = "Asset Category"
	tfCurrency          = "Currency"
	tfSymbol            = "Symbol"
	tfDateTime          = "Date/Time"
	tfQuantity          = "Quantity"
	tfTPrice            = "T. Price"
	tfCPrice            = "C. Price"
	tfProceeds          = "Proceeds"
	tfCommFee           = "Comm/Fee"
	tfCommInCHF         = "Comm in CHF"
	tfMTMInCHF          = "MTM in CHF"
	tfCode              = "Code"
)

const stocksHeldWithIBUK = "Stocks - Held with Interactive Brokers (U.K.) Limited carried by Interactive Brokers LLC"

func (p *parser) parseTrade(r map[string]string) (bool, error) {
	if !(r[lSection] == "Trades" &&
		r[lHeader] == "Data" &&
		r[tfDataDiscriminator] == "Order" &&
		(r[tfAssetCategory] == "Stocks" ||
			r[tfAssetCategory] == stocksHeldWithIBUK)) {
		return false, nil
	}
	var (
		currency, stock           *model.Commodity
		date                      time.Time
		desc                      string
		qty, price, proceeds, fee decimal.Decimal
		err                       error
	)
	if currency, err = p.registry.Commodities().Get(r[tfCurrency]); err != nil {
		return false, err
	}
	if stock, err = p.registry.Commodities().Get(r[tfSymbol]); err != nil {
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
	p.builder.Add(transaction.Builder{
		Date:        date,
		Description: desc,
		Postings: posting.Builders{
			{
				Credit:    p.trading,
				Debit:     p.account,
				Commodity: stock,
				Quantity:  qty,
			},
			{
				Credit:    p.trading,
				Debit:     p.account,
				Commodity: currency,
				Quantity:  proceeds,
			},
			{
				Credit:    p.fee,
				Debit:     p.account,
				Commodity: currency,
				Quantity:  fee,
			},
		}.Build(),
		Targets: []*model.Commodity{stock, currency},
	}.Build())
	return true, nil
}

const forexHeldWithIBUK = "Forex - Held with Interactive Brokers (U.K.) Limited carried by Interactive Brokers LLC"

func (p *parser) parseForex(r map[string]string) (bool, error) {
	if !(r[lSection] == "Trades" &&
		r[lHeader] == "Data" &&
		r[tfDataDiscriminator] == "Order" &&
		(r[tfAssetCategory] == "Forex" ||
			r[tfAssetCategory] == forexHeldWithIBUK)) {
		return false, nil
	}
	if p.baseCurrency == nil {
		return false, fmt.Errorf("base currency is not defined")
	}
	var (
		currency, stock           *model.Commodity
		date                      time.Time
		desc                      string
		qty, price, proceeds, fee decimal.Decimal
		err                       error
	)
	if currency, err = p.registry.Commodities().Get(r[tfCurrency]); err != nil {
		return false, err
	}
	if stock, err = p.registry.Commodities().Get(strings.SplitN(r[tfSymbol], ".", 2)[0]); err != nil {
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
	if fee, err = parseRoundedDecimal(r[tfCommInCHF]); err != nil {
		return false, err
	}
	if qty.IsPositive() {
		desc = fmt.Sprintf("Buy %s %s @ %s %s", qty, stock.Name(), price, currency.Name())
	} else {
		desc = fmt.Sprintf("Sell %s %s @ %s %s", qty, stock.Name(), price, currency.Name())
	}
	postings := posting.Builders{
		{
			Credit:    p.trading,
			Debit:     p.account,
			Commodity: stock,
			Quantity:  qty,
		},
		{
			Credit:    p.trading,
			Debit:     p.account,
			Commodity: currency,
			Quantity:  proceeds,
		},
	}
	if !fee.IsZero() {
		postings = append(postings, posting.Builder{
			Credit:    p.fee,
			Debit:     p.account,
			Commodity: p.baseCurrency,
			Quantity:  fee,
		})
	}
	p.builder.Add(transaction.Builder{
		Date:        date,
		Description: desc,
		Postings:    postings.Build(),
		Targets:     []*model.Commodity{stock, currency},
	}.Build())
	return true, nil
}

const (
	dwfCurrency    = "Currency"
	dwfSettleDate  = "Settle Date"
	dwfDescription = "Description"
	dwfAmount      = "Amount"
)

func (p *parser) parseDepositOrWithdrawal(r map[string]string) (bool, error) {
	if !(r[lSection] == "Deposits & Withdrawals" &&
		r[lHeader] == "Data" &&
		r[dwfCurrency] != "Total" &&
		r[dwfSettleDate] != "") {
		return false, nil
	}
	var (
		currency *model.Commodity
		date     time.Time
		desc     string
		quantity decimal.Decimal
		err      error
	)
	if currency, err = p.registry.Commodities().Get(r[dwfCurrency]); err != nil {
		return false, err
	}
	if date, err = parseDate(r[dwfSettleDate]); err != nil {
		return false, err
	}
	if quantity, err = parseRoundedDecimal(r[dwfAmount]); err != nil {
		return false, err
	}
	if quantity.IsPositive() {
		desc = fmt.Sprintf("Deposit %s %s", quantity, currency.Name())
	} else {
		desc = fmt.Sprintf("Withdraw %s %s", quantity, currency.Name())
	}
	p.builder.Add(transaction.Builder{
		Date:        date,
		Description: desc,
		Postings: posting.Builder{
			Credit:    p.transfer,
			Debit:     p.account,
			Commodity: currency,
			Quantity:  quantity,
		}.Build(),
	}.Build())
	return true, nil
}

const (
	dfCurrency    = "Currency"
	dfDate        = "Date"
	dfDescription = "Description"
	dfAmount      = "Amount"
)

func (p *parser) parseDividend(r map[string]string) (bool, error) {
	if !(r[lSection] == "Dividends" &&
		r[lHeader] == "Data" &&
		!strings.HasPrefix(r[dfCurrency], "Total")) {
		return false, nil
	}
	var (
		currency, security *model.Commodity
		date               time.Time
		desc               = r[dfDescription]
		quantity           decimal.Decimal
		symbol             string
		err                error
	)
	if currency, err = p.registry.Commodities().Get(r[dfCurrency]); err != nil {
		return false, err
	}
	if date, err = parseDate(r[dfDate]); err != nil {
		return false, err
	}
	if quantity, err = parseDecimal(r[dfAmount]); err != nil {
		return false, err
	}
	if symbol, err = parseDividendSymbol(r[dfDescription]); err != nil {
		return false, err
	}
	if security, err = p.registry.Commodities().Get(symbol); err != nil {
		return false, err
	}
	p.builder.Add(transaction.Builder{
		Date:        date,
		Description: desc,
		Postings: posting.Builder{
			Credit:    p.dividend,
			Debit:     p.account,
			Commodity: currency,
			Quantity:  quantity,
		}.Build(),
		Targets: []*model.Commodity{security},
	}.Build())
	return true, nil
}

var dividendSymbolRegex = regexp.MustCompile("[A-Za-z0-9]+")

func parseDividendSymbol(s string) (string, error) {
	symbol := dividendSymbolRegex.FindString(s)
	if symbol == "" {
		return symbol, fmt.Errorf("invalid symbol name %s", s)
	}
	return symbol, nil
}

const (
	wtfCurrency    = "Currency"
	wtfDate        = "Date"
	wtfDescription = "Description"
	wtfAmount      = "Amount"
	wtfCode        = "Code"
)

func (p *parser) parseWithholdingTax(r map[string]string) (bool, error) {
	if !(r[lSection] == "Withholding Tax" &&
		r[lHeader] == "Data" &&
		!strings.HasPrefix(r[wtfCurrency], "Total")) {
		return false, nil
	}
	var (
		desc               = r[wtfDescription]
		currency, security *model.Commodity
		date               time.Time
		quantity           decimal.Decimal
		symbol             string
		err                error
	)
	if currency, err = p.registry.Commodities().Get(r[wtfCurrency]); err != nil {
		return false, err
	}
	if date, err = parseDate(r[wtfDate]); err != nil {
		return false, err
	}
	if quantity, err = parseDecimal(r[wtfAmount]); err != nil {
		return false, err
	}
	if symbol, err = parseDividendSymbol(r[wtfDescription]); err != nil {
		return false, err
	}
	if security, err = p.registry.Commodities().Get(symbol); err != nil {
		return false, err
	}
	p.builder.Add(transaction.Builder{
		Date:        date,
		Description: desc,
		Postings: posting.Builder{
			Credit:    p.tax,
			Debit:     p.account,
			Commodity: currency,
			Quantity:  quantity,
		}.Build(),
		Targets: []*model.Commodity{security},
	}.Build())
	return true, nil
}

const (
	ffSubtitle    = "Subtitle"
	ffCurrency    = "Currency"
	ffDate        = "Date"
	ffDescription = "Description"
	ffAmount      = "Amount"
)

func (p *parser) parseFees(r map[string]string) (bool, error) {
	if !(r[lSection] == "Fees" &&
		r[lHeader] == "Data" &&
		r[ffSubtitle] == "Other Fees" &&
		!strings.HasPrefix(r[ffCurrency], "Total")) {
		return false, nil
	}
	var (
		desc     = r[ffDescription]
		currency *model.Commodity
		date     time.Time
		amount   decimal.Decimal
		err      error
	)
	if currency, err = p.registry.Commodities().Get(r[ffCurrency]); err != nil {
		return false, err
	}
	if date, err = parseDate(r[ffDate]); err != nil {
		return false, err
	}
	if amount, err = parseDecimal(r[ffAmount]); err != nil {
		return false, err
	}
	p.builder.Add(transaction.Builder{
		Date:        date,
		Description: desc,
		Postings: posting.Builder{
			Credit:    p.fee,
			Debit:     p.account,
			Commodity: currency,
			Quantity:  amount,
		}.Build(),
	}.Build())
	return true, nil
}

// Interest,Data,USD,2020-07-06,USD Debit Interest for Jun-2020,-0.73
func (p *parser) parseInterest(r map[string]string) (bool, error) {
	if !(r[lSection] == "Interest" &&
		r[lHeader] == "Data" &&
		!strings.HasPrefix(r[dfCurrency], "Total")) {
		return false, nil
	}
	var (
		currency *model.Commodity
		date     time.Time
		quantity decimal.Decimal
		desc     = r[dfDescription]
		err      error
	)
	if currency, err = p.registry.Commodities().Get(r[dfCurrency]); err != nil {
		return false, err
	}
	if date, err = parseDate(r[dfDate]); err != nil {
		return false, err
	}
	if quantity, err = parseDecimal(r[dfAmount]); err != nil {
		return false, err
	}
	p.builder.Add(transaction.Builder{
		Date:        date,
		Description: desc,
		Postings: posting.Builder{
			Credit:    p.interest,
			Debit:     p.account,
			Commodity: currency,
			Quantity:  quantity,
		}.Build(),
		Targets: []*model.Commodity{currency},
	}.Build())
	return true, nil
}

const (
	opfDataDiscriminator = "DataDiscriminator"
	opfAssetCategory     = "Asset Category"
	opfCurrency          = "Currency"
	opfSymbol            = "Symbol"
	opfQuantity          = "Quantity"
	opfMult              = "Mult"
	opfCostPrice         = "Cost Price"
	opfCostBasis         = "Cost Basis"
	opfClosePrice        = "Close Price"
	opfValue             = "Value"
	opfUnrealizedPL      = "Unrealized P/L"
	opfUnrealizedPLPct   = "Unrealized P/L %"
	opfCode              = "Code"
)

func (p *parser) createAssertions(r map[string]string) (bool, error) {
	if !(r[lSection] == "Open Positions" &&
		r[lHeader] == "Data" &&
		r[opfDataDiscriminator] == "Summary") {
		return false, nil
	}
	if p.dateTo.IsZero() {
		return false, fmt.Errorf("report end date has not been parsed yet")
	}
	var (
		symbol   *model.Commodity
		quantity decimal.Decimal
		err      error
	)
	if symbol, err = p.registry.Commodities().Get(r[opfSymbol]); err != nil {
		return false, err
	}
	if quantity, err = decimal.NewFromString(r[opfQuantity]); err != nil {
		return false, err
	}
	p.builder.Add(&model.Assertion{
		Date: p.dateTo,
		Balances: []model.Balance{
			{
				Account:   p.account,
				Commodity: symbol,
				Quantity:  quantity,
			},
		},
	})
	return true, nil
}

const (
	fbfAssetCategory     = "Asset Category"
	fbfCurrency          = "Currency"
	fbfDescription       = "Description"
	fbfQuantity          = "Quantity"
	fbfCostPrice         = "Cost Price"
	fbfCostBasisInCHF    = "Cost Basis in CHF"
	fbfClosePrice        = "Close Price"
	fbfValueInCHF        = "Value in CHF"
	fbfUnrealizedPLInCHF = "Unrealized P/L in CHF"
	fbfCode              = "Code"
)

func (p *parser) createCurrencyAssertions(r map[string]string) (bool, error) {
	if !(r[lSection] == "Forex Balances" &&
		r[lHeader] == "Data" &&
		(r[fbfAssetCategory] == "Forex" ||
			r[fbfAssetCategory] == forexHeldWithIBUK)) {
		return false, nil
	}
	if p.dateTo.IsZero() {
		return false, fmt.Errorf("report end date has not been parsed yet")
	}
	var (
		symbol *model.Commodity
		amount decimal.Decimal
		err    error
	)
	if symbol, err = p.registry.Commodities().Get(r[fbfDescription]); err != nil {
		return false, err
	}
	if amount, err = parseRoundedDecimal(r[fbfQuantity]); err != nil {
		return false, err
	}
	p.builder.Add(&model.Assertion{
		Date: p.dateTo,
		Balances: []model.Balance{
			{
				Account:   p.account,
				Commodity: symbol,
				Quantity:  amount,
			},
		},
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
