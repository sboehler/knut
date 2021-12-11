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
	"time"

	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"
	"golang.org/x/text/encoding/charmap"

	"github.com/sboehler/knut/cmd/flags"
	"github.com/sboehler/knut/cmd/importer"
	"github.com/sboehler/knut/lib/ledger"
	"github.com/sboehler/knut/lib/printer"
)

// CreateCmd creates the cobra command.
func CreateCmd() *cobra.Command {

	var r runner

	var cmd = &cobra.Command{
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
		ctx    = ledger.NewContext()
		err    error
	)
	if reader, err = flags.OpenFile(args[0]); err != nil {
		return err
	}
	var p = Parser{
		reader:  csv.NewReader(charmap.ISO8859_1.NewDecoder().Reader(reader)),
		builder: ledger.NewBuilder(ctx, ledger.Filter{}),
	}
	if p.account, err = r.accountFlag.Value(ctx); err != nil {
		return err
	}
	if err = p.parse(); err != nil {
		return err
	}
	out := bufio.NewWriter(cmd.OutOrStdout())
	defer out.Flush()
	_, err = printer.New().PrintLedger(out, p.builder.Build())
	return err
}

func init() {
	importer.Register(CreateCmd)
}

// Parser is a parser for account statements
type Parser struct {
	reader  *csv.Reader
	account *ledger.Account
	builder *ledger.Builder

	currency *ledger.Commodity
}

func (p *Parser) parse() error {
	p.reader.FieldsPerRecord = -1
	p.reader.LazyQuotes = true
	p.reader.TrimLeadingSpace = true
	p.reader.Comma = ';'
	for {
		var err = p.readLine()
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
	var currencyHeaders = map[string]bool{
		"Währung:":  true,
		"Currency:": true,
	}
	var err error
	if currencyHeaders[l[hfHeader]] {
		if p.currency, err = p.builder.Context.GetCommodity(l[hfData]); err != nil {
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
	if date, err = time.Parse("2006-01-02", l[bfBuchungsdatum]); err != nil {
		return err
	}
	if amount, err = parseAmount(l); err != nil {
		return err
	}
	p.builder.AddTransaction(ledger.Transaction{
		Date:        date,
		Description: l[bfAvisierungstext],
		Postings: []ledger.Posting{
			ledger.NewPosting(p.builder.Context.TBDAccount(), p.account, p.currency, amount),
		},
	})
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
