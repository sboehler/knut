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

package postfinance

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

// CreateCmd creates the cobra command.
func CreateCmd() *cobra.Command {

	var r runner

	cmd := &cobra.Command{
		Use:   "ch.postfinance",
		Short: "Import Postfinance CSV account statements",

		Args: cobra.ExactValidArgs(1),

		RunE: r.run,
	}
	r.setupFlags(cmd)
	return cmd
}

type runner struct {
	accountFlag flags.AccountFlag
}

func (r *runner) setupFlags(cmd *cobra.Command) {
	cmd.Flags().VarP(&r.accountFlag, "account", "a", "account name")
	cmd.MarkFlagRequired("account")
}

func (r *runner) run(cmd *cobra.Command, args []string) error {
	var (
		reader *bufio.Reader
		ctx    = journal.NewContext()
		err    error
	)
	if reader, err = flags.OpenFile(args[0]); err != nil {
		return err
	}
	p := Parser{
		reader:  csv.NewReader(reader),
		journal: journal.New(ctx),
	}
	if p.account, err = r.accountFlag.Value(ctx); err != nil {
		return err
	}
	if err = p.parse(); err != nil {
		return err
	}
	out := bufio.NewWriter(cmd.OutOrStdout())
	defer out.Flush()
	_, err = journal.NewPrinter().PrintLedger(out, p.journal.SortedDays())
	return err
}

func init() {
	importer.Register(CreateCmd)
}

// Parser is a parser for account statements
type Parser struct {
	reader  *csv.Reader
	account *journal.Account
	journal *journal.Journal

	currency *journal.Commodity
}

func (p *Parser) parse() error {
	p.reader.FieldsPerRecord = -1
	p.reader.LazyQuotes = true
	p.reader.TrimLeadingSpace = true
	p.reader.Comma = ';'
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

func (p *Parser) readLine() error {
	var (
		line []string
		err  error
	)
	if line, err = p.reader.Read(); err != nil {
		return err
	}
	switch len(line) {
	case 2:
		return p.readHeaderLine(line)
	case 6:
		return p.readBookingLine(line)
	default:
		return nil
	}
}

type headerField int

const (
	hfHeader headerField = iota
	hfData
)

func (p *Parser) readHeaderLine(l []string) error {
	currencyHeaders := set.Of("Währung:", "Currency:")
	var err error
	if currencyHeaders.Has(l[hfHeader]) {
		sym := strings.Trim(l[hfData], "=\"")
		if p.currency, err = p.journal.Context.GetCommodity(sym); err != nil {
			return err
		}
	}
	return nil
}

type bookingField int

const (
	bfBuchungsdatum bookingField = iota
	bfAvisierungstext
	bfGutschriftInCHF
	bfLastschriftInCHF
	bfValuta
	bfSaldoInCHF
)

func (p *Parser) readBookingLine(l []string) error {
	if isHeader(l) {
		return nil
	}
	var (
		date   time.Time
		amount decimal.Decimal
		err    error
	)
	if date, err = time.Parse("02.01.2006", l[bfBuchungsdatum]); err != nil {
		return err
	}
	if amount, err = parseAmount(l); err != nil {
		return err
	}
	p.journal.AddTransaction(journal.TransactionBuilder{
		Date:        date,
		Description: strings.TrimSpace(l[bfAvisierungstext]),
		Postings: journal.PostingBuilder{
			Credit:    p.journal.Context.TBDAccount(),
			Debit:     p.account,
			Commodity: p.currency,
			Amount:    amount,
		}.Singleton(),
	}.Build())
	return nil
}

func parseAmount(l []string) (decimal.Decimal, error) {
	var (
		amount decimal.Decimal
		field  bookingField
	)
	switch {
	case len(l[bfGutschriftInCHF]) > 0 && len(l[bfLastschriftInCHF]) == 0:
		field = bfGutschriftInCHF
	case len(l[bfGutschriftInCHF]) == 0 && len(l[bfLastschriftInCHF]) > 0:
		field = bfLastschriftInCHF
	default:
		return amount, fmt.Errorf("invalid amount fields %q %q", l[bfGutschriftInCHF], l[bfLastschriftInCHF])
	}
	return decimal.NewFromString(l[field])
}

func isHeader(s []string) bool {
	return s[bfBuchungsdatum] == "Buchungsdatum"
}
