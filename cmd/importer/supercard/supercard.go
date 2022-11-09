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

package supercard

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
	"golang.org/x/text/encoding/charmap"

	"github.com/sboehler/knut/cmd/flags"
	"github.com/sboehler/knut/cmd/importer"
	"github.com/sboehler/knut/lib/journal"
)

// CreateCmd creates the command.
func CreateCmd() *cobra.Command {
	var r runner
	var cmd = &cobra.Command{
		Use:   "ch.supercard",
		Short: "Import Supercard credit card statements",
		Long:  `Download the CSV file from their account management tool.`,

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
	account flags.AccountFlag
}

func (r *runner) setupFlags(cmd *cobra.Command) {
	cmd.Flags().VarP(&r.account, "account", "a", "account name")
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
	var p = parser{
		reader:  csv.NewReader(charmap.ISO8859_1.NewDecoder().Reader(f)),
		builder: journal.New(ctx),
	}

	if p.account, err = r.account.Value(ctx); err != nil {
		return err
	}
	if err = p.parse(); err != nil {
		return err
	}
	out := bufio.NewWriter(cmd.OutOrStdout())
	defer out.Flush()
	_, err = journal.NewPrinter().PrintLedger(out, p.builder.SortedDays())
	return err
}

type parser struct {
	reader  *csv.Reader
	account *journal.Account
	builder *journal.Journal
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
	if len(r) == 11 || r[fieldKontonummer] == "" {
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
		commodity *journal.Commodity
		date      time.Time
		amount    decimal.Decimal
		err       error
	)
	if date, err = p.parseDate(r); err != nil {
		return fmt.Errorf("%v %w", r, err)
	}
	if amount, err = p.parseAmount(r); err != nil {
		return err
	}
	if commodity, err = p.builder.Context.GetCommodity(currency); err != nil {
		return err
	}
	p.builder.AddTransaction(journal.TransactionBuilder{
		Date:        date,
		Description: words,
		Postings: []*journal.Posting{
			journal.NewPosting(p.builder.Context.TBDAccount(), p.account, commodity, amount),
		},
	}.Build())
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
