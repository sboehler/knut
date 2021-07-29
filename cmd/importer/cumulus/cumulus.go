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

package cumulus

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
		Use:   "ch.cumulus",
		Short: "Import Cumulus credit card statements",
		Long: `Download a PDF account statement and run it through tabula (https://tabula.technology/),
using the default options and saving it to CSV. This importer will parse the unaltered CSV.`,

		Args: cobra.ExactValidArgs(1),

		RunE: run,
	}
	cmd.Flags().StringP("account", "a", "", "account name")
	return &cmd

}

func init() {
	importer.Register(CreateCmd)
}

func run(cmd *cobra.Command, args []string) error {
	account, err := flags.GetAccountFlag(cmd, "account")
	if err != nil {
		return err
	}
	f, err := os.Open(args[0])
	if err != nil {
		return err
	}
	reader := csv.NewReader(bufio.NewReader(f))
	p := parser{
		reader:  reader,
		account: account,
		builder: ledger.NewBuilder(ledger.Filter{}),
	}
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
	account *accounts.Account
	builder *ledger.Builder
	last    *ledger.Transaction
}

func (p *parser) parse() error {
	p.reader.FieldsPerRecord = -1
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
	r, err := p.reader.Read()
	fmt.Println(r)
	if err != nil {
		return err
	}
	if ok, err := p.parseRounding(r); ok || err != nil {
		return err
	}
	if ok, err := p.parseFXComment(r); ok || err != nil {
		return err
	}
	if ok, err := p.parseBooking(r); ok || err != nil {
		return err
	}
	return nil
}

var dateRegex = regexp.MustCompile(`\d\d.\d\d.\d\d\d\d`)

// bookingField denotes the labels the fields of a regular booking line.
type bookingField int

const (
	bfEinkaufsDatum bookingField = iota
	bfVerbuchtAm
	bfBeschreibung
	bfGutschriftCHF
	bfBelastungCHF
)

func (p *parser) parseBooking(r []string) (bool, error) {
	if !checkValidBookingLine(r) {
		return false, nil
	}
	if len(r) != 5 {
		return false, fmt.Errorf("expected five items, got %v", r)
	}
	var (
		err    error
		desc   = r[bfBeschreibung]
		amount decimal.Decimal
		chf    *commodities.Commodity
		date   time.Time
	)
	if date, err = time.Parse("02.01.2006", r[bfEinkaufsDatum]); err != nil {
		return false, err
	}
	if amount, err = parseAmount(r[bfBelastungCHF], r[bfGutschriftCHF]); err != nil {
		return false, err
	}
	if chf, err = commodities.Get("CHF"); err != nil {
		return false, err
	}
	p.last = &ledger.Transaction{
		Date:        date,
		Description: desc,
		Postings: []*ledger.Posting{
			ledger.NewPosting(accounts.TBDAccount(), p.account, chf, amount),
		},
	}
	p.builder.AddTransaction(p.last)
	return true, nil
}

func parseAmount(creditField, debitField string) (decimal.Decimal, error) {
	var (
		sign   = decimal.NewFromInt(1)
		field  string
		amount decimal.Decimal
		err    error
	)
	switch {
	case len(creditField) > 0 && len(debitField) == 0:
		field = creditField
		sign = sign.Neg()
	case len(creditField) == 0 && len(debitField) > 0:
		field = debitField
	default:
		return amount, fmt.Errorf("row has invalid amounts: %v %v", creditField, debitField)
	}
	if amount, err = parseDecimal(field); err != nil {
		return amount, err
	}
	return amount.Mul(sign), nil
}

func checkValidBookingLine(r []string) bool {
	return dateRegex.MatchString(r[0]) && dateRegex.MatchString(r[1])
}

func (p *parser) parseFXComment(r []string) (bool, error) {
	if !(len(r) == 5 &&
		len(r[bfEinkaufsDatum]) == 0 &&
		len(r[bfVerbuchtAm]) == 0 &&
		len(r[bfBeschreibung]) > 0 &&
		len(r[bfGutschriftCHF]) == 0 &&
		len(r[bfBelastungCHF]) == 0) {
		return false, nil
	}
	if p.last == nil {
		return false, fmt.Errorf("fx comment but no previous transaction")
	}
	p.last.Description = fmt.Sprintf("%s %s", p.last.Description, r[bfBeschreibung])
	return true, nil
}

// roundingField denotes the labels the fields of a "Rundungskorrektur" line.
type roundingField int

const (
	rfEinkaufsDatum roundingField = iota
	rfBeschreibung
	rfGutschriftCHF
	rfBelastungCHF
)

func (p *parser) parseRounding(r []string) (bool, error) {
	if !(dateRegex.MatchString(r[rfEinkaufsDatum]) && r[rfBeschreibung] == "Rundungskorrektur") {
		return false, nil
	}
	if len(r) != 4 {
		return false, fmt.Errorf("expected three items, got %v", r)
	}
	var (
		err    error
		amount decimal.Decimal
		date   time.Time
		chf    *commodities.Commodity
	)
	if date, err = time.Parse("02.01.2006", r[rfEinkaufsDatum]); err != nil {
		return false, err
	}
	if amount, err = parseAmount(r[rfBelastungCHF], r[rfGutschriftCHF]); err != nil {
		return false, err
	}
	if chf, err = commodities.Get("CHF"); err != nil {
		return false, err
	}
	p.builder.AddTransaction(&ledger.Transaction{
		Date:        date,
		Description: r[rfBeschreibung],
		Postings: []*ledger.Posting{
			ledger.NewPosting(accounts.TBDAccount(), p.account, chf, amount),
		},
	})
	return true, nil
}

func parseDecimal(s string) (decimal.Decimal, error) {
	return decimal.NewFromString(strings.ReplaceAll(s, "'", ""))
}
