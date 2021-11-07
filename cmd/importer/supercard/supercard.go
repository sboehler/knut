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

package supercard

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
		Use:   "ch.supercard",
		Short: "Import Supercard credit card statements",
		Long:  `Download the CSV file from their account management tool.`,

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
	var (
		reader = csv.NewReader(bufio.NewReader(charmap.ISO8859_1.NewDecoder().Reader(f)))
		p      = parser{
			reader:  reader,
			account: account,
			builder: ledger.NewBuilder(nil, nil),
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
	account *accounts.Account
	builder *ledger.Builder
}

func (p *parser) parse() error {
	p.reader.TrimLeadingSpace = true
	p.reader.Comma = ';'
	p.reader.FieldsPerRecord = 13
	if err := p.checkFirstLine(); err != nil {
		return err
	}
	if err := p.skipHeader(); err != nil {
		return err
	}
	p.reader.FieldsPerRecord = -1
	for {
		if err := p.readLine(); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}

func (p *parser) checkFirstLine() error {
	var fpr = p.reader.FieldsPerRecord
	defer func() {
		p.reader.FieldsPerRecord = fpr
	}()
	p.reader.FieldsPerRecord = 2
	rec, err := p.reader.Read()
	if err != nil {
		return err
	}
	if rec[0] != "sep=" || rec[1] != "" {
		return fmt.Errorf("unexpected first line %q", rec)
	}
	return nil
}

func (p *parser) skipHeader() error {
	_, err := p.reader.Read()
	return err
}

func (p *parser) readLine() error {
	r, err := p.reader.Read()
	if err != nil {
		return err
	}
	if r[fieldBuchungstext] == "Saldovortrag" {
		return nil
	}
	if len(r) == 11 {
		return nil
	}
	if len(r) != 13 {
		return fmt.Errorf("record %v with invalid length %d", r, len(r))
	}
	if err := p.parseBooking(r); err != nil {
		return err
	}
	return nil
}

type field int

const (
	fieldKontonummer field = iota
	fieldKartennummer
	fieldKontoKarteninhaber
	fieldEinkaufsdatum
	fieldBuchungstext
	fieldBranche
	fieldBetrag
	fieldOriginalwährung
	fieldKurs
	fieldWährung
	fieldBelastung
	fieldGutschrift
	fieldBuchung
)

func (p *parser) parseBooking(r []string) error {
	var (
		words     = p.parseWords(r)
		currency  = p.parseCurrency(r)
		commodity *commodities.Commodity
		date      time.Time
		amount    decimal.Decimal
		err       error
	)
	if date, err = p.parseDate(r); err != nil {
		return err
	}
	if amount, err = p.parseAmount(r); err != nil {
		return err
	}
	if commodity, err = commodities.Get(currency); err != nil {
		return err
	}
	p.builder.AddTransaction(&ledger.Transaction{
		Date:        date,
		Description: words,
		Postings: []ledger.Posting{
			ledger.NewPosting(accounts.TBDAccount(), p.account, commodity, amount),
		},
	})
	return nil
}

func (p *parser) parseCurrency(r []string) string {
	return r[fieldWährung]
}

var space = regexp.MustCompile(`\s+`)

func (p *parser) parseWords(r []string) string {
	var words = strings.Join([]string{r[fieldBuchungstext], r[fieldBranche]}, " ")
	return space.ReplaceAllString(words, " ")
}

func (p *parser) parseDate(r []string) (time.Time, error) {
	return time.Parse("02.01.2006", r[fieldEinkaufsdatum])
}

func (p *parser) parseAmount(r []string) (decimal.Decimal, error) {
	var (
		sign  = decimal.NewFromInt(1)
		field field
		res   decimal.Decimal
	)
	switch {
	case len(r[fieldGutschrift]) > 0:
		field = fieldGutschrift
	case len(r[fieldBelastung]) > 0:
		field = fieldBelastung
		sign = sign.Neg()
	default:
		return res, fmt.Errorf("empty amount fields: %s %s", r[fieldGutschrift], r[fieldBelastung])
	}
	amt, err := decimal.NewFromString(r[field])
	if err != nil {
		return res, err
	}
	return amt.Mul(sign), nil
}
