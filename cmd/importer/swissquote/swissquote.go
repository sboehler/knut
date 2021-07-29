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

package swissquote

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"
	"golang.org/x/text/encoding/charmap"

	"github.com/sboehler/knut/cmd/flags"
	"github.com/sboehler/knut/cmd/importer"
	"github.com/sboehler/knut/lib/ledger"
	"github.com/sboehler/knut/lib/model/accounts"
	"github.com/sboehler/knut/lib/model/commodities"
	"github.com/sboehler/knut/lib/printer"
)

// CreateCmd creates the command.
func CreateCmd() *cobra.Command {
	var cmd = cobra.Command{
		Use:   "ch.swissquote",
		Short: "Import Swissquote account reports",
		Long:  `Parses CSV files from Swissquote's transactions overview.`,

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
	if o.account, err = flags.GetAccountFlag(cmd, "account"); err != nil {
		return err
	}
	if o.dividend, err = flags.GetAccountFlag(cmd, "dividend"); err != nil {
		return err
	}
	if o.interest, err = flags.GetAccountFlag(cmd, "interest"); err != nil {
		return err
	}
	if o.tax, err = flags.GetAccountFlag(cmd, "tax"); err != nil {
		return err
	}
	if o.fee, err = flags.GetAccountFlag(cmd, "fee"); err != nil {
		return err
	}
	f, err := os.Open(args[0])
	if err != nil {
		return err
	}
	var (
		reader = csv.NewReader(charmap.ISO8859_1.NewDecoder().Reader(f))
		p      = parser{
			reader:  reader,
			options: o,
			builder: ledger.NewBuilder(ledger.Filter{}),
		}
	)
	if err = p.parse(); err != nil {
		return err
	}
	w := bufio.NewWriter(cmd.OutOrStdout())
	defer w.Flush()
	_, err = printer.New().PrintLedger(w, p.builder.Build())
	return err
}

type parser struct {
	reader  *csv.Reader
	options options
	builder *ledger.Builder
	last    *record
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
	r, err := lineToRecord(l)
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

func lineToRecord(l []string) (*record, error) {
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
		if r.symbol, err = commodities.Get(l[fSymbol]); err != nil {
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
	if r.currency, err = commodities.Get(l[fWährung]); err != nil {
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
	currency, symbol                                   *commodities.Commodity
}

func (p *parser) parseTrade(r *record) (bool, error) {
	if !(r.trxType == "Kauf" || r.trxType == "Verkauf") {
		return false, nil
	}
	var (
		proceeds = r.netAmount.Add(r.fee)
		fee      = r.fee.Neg()
		qty      = r.quantity
		desc     = fmt.Sprintf("%s %s %s x %s %s %s @ %s %s", r.orderNo, r.trxType, r.quantity, r.symbol, r.name, r.isin, r.price, r.currency)
	)
	if proceeds.IsPositive() {
		qty = qty.Neg()
	}
	p.builder.AddTransaction(&ledger.Transaction{
		Date:        r.date,
		Description: desc,
		Postings: []*ledger.Posting{
			ledger.NewPosting(accounts.EquityAccount(), p.options.account, r.symbol, qty),
			ledger.NewPosting(accounts.EquityAccount(), p.options.account, r.currency, proceeds),
			ledger.NewPosting(p.options.fee, p.options.account, r.currency, fee),
		},
	})
	return true, nil
}

func (p *parser) parseForex(r *record) (bool, error) {
	var w = map[string]bool{
		"Forex-Gutschrift":    true,
		"Forex-Belastung":     true,
		"Fx-Gutschrift Comp.": true,
		"Fx-Belastung Comp.":  true,
	}
	if _, ok := w[r.trxType]; !ok {
		if p.last != nil {
			return false, fmt.Errorf("expected forex transaction, got %v", r)
		}
		return false, nil
	}
	if p.last == nil {
		p.last = r
		return true, nil
	}
	var desc = fmt.Sprintf("%s %s %s / %s %s %s", p.last.trxType, p.last.netAmount, p.last.currency, r.trxType, r.netAmount, r.currency)
	p.builder.AddTransaction(&ledger.Transaction{
		Date:        r.date,
		Description: desc,
		Postings: []*ledger.Posting{
			ledger.NewPosting(accounts.EquityAccount(), p.options.account, p.last.currency, p.last.netAmount),
			ledger.NewPosting(accounts.EquityAccount(), p.options.account, r.currency, r.netAmount),
		},
	})
	p.last = nil
	return true, nil
}

func (p *parser) parseDividend(r *record) (bool, error) {
	var w = map[string]bool{
		"Capital Gain":       true,
		"Kapitalrückzahlung": true,
		"Dividende":          true,
	}
	if _, ok := w[r.trxType]; !ok {
		return false, nil
	}
	var postings = []*ledger.Posting{
		ledger.NewPosting(p.options.dividend, p.options.account, r.currency, r.price),
	}
	if !r.fee.IsZero() {
		postings = append(postings, ledger.NewPosting(p.options.account, p.options.tax, r.currency, r.fee))
	}
	p.builder.AddTransaction(&ledger.Transaction{
		Date:        r.date,
		Description: fmt.Sprintf("%s %s %s %s", r.trxType, r.symbol, r.name, r.isin),
		Postings:    postings,
	})
	return true, nil
}

func (p *parser) parseCustodyFees(r *record) (bool, error) {
	if r.trxType != "Depotgebühren" {
		return false, nil
	}
	p.builder.AddTransaction(&ledger.Transaction{
		Date:        r.date,
		Description: r.trxType,
		Postings: []*ledger.Posting{
			ledger.NewPosting(p.options.fee, p.options.account, r.currency, r.netAmount),
		},
	})
	return true, nil
}

func (p *parser) parseMoneyTransfer(r *record) (bool, error) {
	var w = map[string]bool{
		"Einzahlung": true,
		"Auszahlung": true,
		"Vergütung":  true,
		"Belastung":  true,
	}
	if _, ok := w[r.trxType]; !ok {
		return false, nil
	}
	p.builder.AddTransaction(&ledger.Transaction{
		Date:        r.date,
		Description: r.trxType,
		Postings: []*ledger.Posting{
			ledger.NewPosting(accounts.TBDAccount(), p.options.account, r.currency, r.netAmount),
		},
	})
	return true, nil
}

func (p *parser) parseInterestIncome(r *record) (bool, error) {
	if r.trxType != "Zins" {
		return false, nil
	}
	p.builder.AddTransaction(&ledger.Transaction{
		Date:        r.date,
		Description: r.trxType,
		Postings: []*ledger.Posting{
			ledger.NewPosting(p.options.interest, p.options.account, r.currency, r.netAmount),
		},
	})
	return true, nil
}

func (p *parser) parseCatchall(r *record) (bool, error) {
	p.builder.AddTransaction(&ledger.Transaction{
		Date:        r.date,
		Description: r.trxType,
		Postings: []*ledger.Posting{
			ledger.NewPosting(accounts.TBDAccount(), p.options.account, r.currency, r.netAmount),
		},
	})
	return true, nil
}
