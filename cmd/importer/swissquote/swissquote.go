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

package swissquote

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"

	"github.com/sboehler/knut/cmd/flags"
	"github.com/sboehler/knut/cmd/importer"
	"github.com/sboehler/knut/lib/common/set"
	"github.com/sboehler/knut/lib/journal"
)

// CreateCmd creates the command.
func CreateCmd() *cobra.Command {
	var r runner
	cmd := &cobra.Command{
		Use:   "ch.swissquote",
		Short: "Import Swissquote account reports",
		Long:  `Parses CSV files from Swissquote's transactions overview.`,

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
	account, dividend, tax, fee, interest, trading flags.AccountFlag
}

func (r *runner) setupFlags(cmd *cobra.Command) {
	cmd.Flags().VarP(&r.account, "account", "a", "account name")
	cmd.Flags().VarP(&r.interest, "interest", "i", "account name of the interest expense account")
	cmd.Flags().VarP(&r.dividend, "dividend", "d", "account name of the dividend account")
	cmd.Flags().VarP(&r.tax, "tax", "w", "account name of the withholding tax account")
	cmd.Flags().VarP(&r.fee, "fee", "f", "account name of the fee account")
	cmd.Flags().VarP(&r.trading, "trading", "t", "account name of the trading gain / loss account")
	cmd.MarkFlagRequired("account")
	cmd.MarkFlagRequired("interest")
	cmd.MarkFlagRequired("dividend")
	cmd.MarkFlagRequired("tax")
	cmd.MarkFlagRequired("fee")
	cmd.MarkFlagRequired("trading")
}

func (r *runner) run(cmd *cobra.Command, args []string) error {
	var (
		ctx = journal.NewContext()
		f   *bufio.Reader
		err error
	)
	if f, err = flags.OpenFile(args[0]); err != nil {
		return err
	}
	p := parser{
		reader:  csv.NewReader(f),
		builder: journal.New(ctx),
	}
	if p.account, err = r.account.Value(ctx); err != nil {
		return err
	}
	if p.dividend, err = r.dividend.Value(ctx); err != nil {
		return err
	}
	if p.interest, err = r.interest.Value(ctx); err != nil {
		return err
	}
	if p.tax, err = r.tax.Value(ctx); err != nil {
		return err
	}
	if p.fee, err = r.fee.Value(ctx); err != nil {
		return err
	}
	if p.trading, err = r.trading.Value(ctx); err != nil {
		return err
	}
	if err = p.parse(); err != nil {
		return err
	}
	out := bufio.NewWriter(cmd.OutOrStdout())
	defer out.Flush()
	_, err = journal.NewPrinter().PrintLedger(out, p.builder.ToLedger())
	return err
}

type parser struct {
	reader  *csv.Reader
	builder *journal.Journal
	last    *record

	account, dividend, tax, fee, interest, trading *journal.Account
}

func (p *parser) parse() error {
	p.reader.LazyQuotes = true
	p.reader.Comma = ';'
	p.reader.FieldsPerRecord = 13
	// skip header
	if _, err := p.reader.Read(); err != nil {
		return err
	}
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
	r, err := p.lineToRecord(l)
	if err != nil {
		return err
	}
	if ok, err := p.parseTrade(r); err != nil || ok {
		return err
	}
	if ok, err := p.parseForex(r); err != nil || ok {
		return err
	}
	if ok, err := p.parseDividend(r); err != nil || ok {
		return err
	}
	if ok, err := p.parseCustodyFees(r); err != nil || ok {
		return err
	}
	if ok, err := p.parseMoneyTransfer(r); err != nil || ok {
		return err
	}
	if ok, err := p.parseInterestIncome(r); err != nil || ok {
		return err
	}
	if ok, err := p.parseCatchall(r); err != nil || ok {
		return err
	}
	return fmt.Errorf("unparsed line: %v", l)
}

type field int

const (
	fDatum field = iota
	fAuftragNo
	fTransaktionen
	fSymbol
	fName
	fISIN
	fAnzahl
	fStückpreis
	fKosten
	fAufgelaufeneZinsen
	fNettobetrag
	fSaldo
	fWährung
)

func (p *parser) lineToRecord(l []string) (*record, error) {
	var (
		r = record{
			orderNo: l[fAuftragNo],
			trxType: l[fTransaktionen],
			name:    l[fName],
			isin:    l[fISIN],
		}
		err error
	)
	if r.date, err = parseDateFromDateTime(l[fDatum]); err != nil {
		return nil, err
	}
	if len(l[fSymbol]) > 0 {
		if r.symbol, err = p.builder.Context.GetCommodity(l[fSymbol]); err != nil {
			return nil, err
		}
	}
	if r.quantity, err = parseDecimal(l[fAnzahl]); err != nil {
		return nil, err
	}
	if r.price, err = parseDecimal(l[fStückpreis]); err != nil {
		return nil, err
	}
	if r.fee, err = parseDecimal(l[fKosten]); err != nil {
		return nil, err
	}
	if r.interest, err = parseDecimal(l[fAufgelaufeneZinsen]); err != nil {
		return nil, err
	}
	if r.netAmount, err = parseDecimal(l[fNettobetrag]); err != nil {
		return nil, err
	}
	if r.balance, err = parseDecimal(l[fSaldo]); err != nil {
		return nil, err
	}
	if r.currency, err = p.builder.Context.GetCommodity(l[fWährung]); err != nil {
		return nil, err
	}
	return &r, nil
}

func parseDecimal(s string) (decimal.Decimal, error) {
	return decimal.NewFromString(strings.ReplaceAll(s, "'", ""))
}

func parseDateFromDateTime(s string) (time.Time, error) {
	return time.Parse("02-01-2006", s[:10])
}

type record struct {
	date                                               time.Time
	orderNo, trxType, name, isin                       string
	quantity, price, fee, interest, netAmount, balance decimal.Decimal
	currency, symbol                                   *journal.Commodity
}

func (p *parser) parseTrade(r *record) (bool, error) {
	if !(r.trxType == "Kauf" || r.trxType == "Verkauf") {
		return false, nil
	}
	var (
		proceeds = r.netAmount.Add(r.fee)
		fee      = r.fee.Neg()
		qty      = r.quantity
		desc     = fmt.Sprintf("%s %s %s x %s %s %s @ %s %s", r.orderNo, r.trxType, r.quantity, r.symbol.Name(), r.name, r.isin, r.price, r.currency.Name())
	)
	if proceeds.IsPositive() {
		qty = qty.Neg()
	}
	p.builder.AddTransaction(journal.TransactionBuilder{
		Date:        r.date,
		Description: desc,
		Postings: []*journal.Posting{
			journal.PostingBuilder{
				Credit:    p.trading,
				Debit:     p.account,
				Commodity: r.symbol,
				Amount:    qty,
				Targets:   []*journal.Commodity{r.symbol, r.currency},
			}.Build(),
			journal.PostingBuilder{
				Credit:    p.trading,
				Debit:     p.account,
				Commodity: r.currency,
				Amount:    proceeds,
				Targets:   []*journal.Commodity{r.symbol, r.currency},
			}.Build(),
			journal.PostingBuilder{
				Credit:    p.fee,
				Debit:     p.account,
				Commodity: r.currency,
				Amount:    fee,
				Targets:   []*journal.Commodity{r.symbol, r.currency},
			}.Build(),
		},
	}.Build())
	return true, nil
}

func (p *parser) parseForex(r *record) (bool, error) {
	w := set.Of(
		"Forex-Gutschrift",
		"Forex-Belastung",
		"Fx-Gutschrift Comp.",
		"Fx-Belastung Comp.",
	)
	if !w.Has(r.trxType) {
		if p.last != nil {
			return false, fmt.Errorf("expected forex transaction, got %v", r)
		}
		return false, nil
	}
	if p.last == nil {
		p.last = r
		return true, nil
	}
	desc := fmt.Sprintf("%s %s %s / %s %s %s", p.last.trxType, p.last.netAmount, p.last.currency.Name(), r.trxType, r.netAmount, r.currency.Name())
	p.builder.AddTransaction(journal.TransactionBuilder{
		Date:        r.date,
		Description: desc,
		Postings: []*journal.Posting{
			journal.PostingBuilder{
				Credit:    p.trading,
				Debit:     p.account,
				Commodity: p.last.currency,
				Amount:    p.last.netAmount, Targets: []*journal.Commodity{p.last.currency, r.currency},
			}.Build(),
			journal.PostingBuilder{
				Credit:    p.trading,
				Debit:     p.account,
				Commodity: r.currency,
				Amount:    r.netAmount, Targets: []*journal.Commodity{p.last.currency, r.currency},
			}.Build(),
		},
	}.Build())
	p.last = nil
	return true, nil
}

func (p *parser) parseDividend(r *record) (bool, error) {
	w := set.Of(
		"Capital Gain",
		"Kapitalrückzahlung",
		"Dividende",
	)
	if !w.Has(r.trxType) {
		return false, nil
	}
	postings := journal.PostingBuilder{
		Credit:    p.dividend,
		Debit:     p.account,
		Commodity: r.currency,
		Amount:    r.price,
		Targets:   []*journal.Commodity{r.symbol},
	}.Singleton()
	if !r.fee.IsZero() {
		postings = append(postings, journal.PostingBuilder{
			Credit:    p.account,
			Debit:     p.tax,
			Commodity: r.currency,
			Amount:    r.fee,
			Targets:   []*journal.Commodity{r.symbol},
		}.Build())
	}
	p.builder.AddTransaction(journal.TransactionBuilder{
		Date:        r.date,
		Description: fmt.Sprintf("%s %s %s %s", r.trxType, r.symbol.Name(), r.name, r.isin),
		Postings:    postings,
	}.Build())
	return true, nil
}

func (p *parser) parseCustodyFees(r *record) (bool, error) {
	if r.trxType != "Depotgebühren" {
		return false, nil
	}
	p.builder.AddTransaction(journal.TransactionBuilder{
		Date:        r.date,
		Description: r.trxType,
		Postings: journal.PostingBuilder{
			Credit:    p.fee,
			Debit:     p.account,
			Commodity: r.currency,
			Amount:    r.netAmount,
			Targets:   make([]*journal.Commodity, 0),
		}.Singleton(),
	}.Build())
	return true, nil
}

func (p *parser) parseMoneyTransfer(r *record) (bool, error) {
	w := set.Of(
		"Einzahlung",
		"Auszahlung",
		"Vergütung",
		"Belastung",
	)
	if !w.Has(r.trxType) {
		return false, nil
	}
	p.builder.AddTransaction(journal.TransactionBuilder{
		Date:        r.date,
		Description: r.trxType,
		Postings: journal.PostingBuilder{
			Credit:    p.builder.Context.TBDAccount(),
			Debit:     p.account,
			Commodity: r.currency,
			Amount:    r.netAmount,
		}.Singleton(),
	}.Build())
	return true, nil
}

func (p *parser) parseInterestIncome(r *record) (bool, error) {
	if r.trxType != "Zins" {
		return false, nil
	}
	p.builder.AddTransaction(journal.TransactionBuilder{
		Date:        r.date,
		Description: r.trxType,
		Postings: journal.PostingBuilder{
			Credit:    p.interest,
			Debit:     p.account,
			Commodity: r.currency,
			Amount:    r.netAmount,
			Targets:   []*journal.Commodity{r.currency},
		}.Singleton(),
	}.Build())
	return true, nil
}

func (p *parser) parseCatchall(r *record) (bool, error) {
	p.builder.AddTransaction(journal.TransactionBuilder{
		Date:        r.date,
		Description: r.trxType,
		Postings: journal.PostingBuilder{
			Credit:    p.builder.Context.TBDAccount(),
			Debit:     p.account,
			Commodity: r.currency,
			Amount:    r.netAmount,
		}.Singleton(),
	}.Build())
	return true, nil
}
