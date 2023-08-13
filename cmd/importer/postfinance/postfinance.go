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
	"os"
	"strings"
	"time"

	"github.com/dimchansky/utfbom"
	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"

	"github.com/sboehler/knut/cmd/flags"
	"github.com/sboehler/knut/cmd/importer"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/model"
	"github.com/sboehler/knut/lib/model/posting"
	"github.com/sboehler/knut/lib/model/registry"
	"github.com/sboehler/knut/lib/model/transaction"
)

// CreateCmd creates the cobra command.
func CreateCmd() *cobra.Command {

	var r runner

	cmd := &cobra.Command{
		Use:   "ch.postfinance",
		Short: "Import Postfinance CSV account statements",

		Args: cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),

		Run: r.run,
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

func (r *runner) run(cmd *cobra.Command, args []string) {
	if err := r.runE(cmd, args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func (r *runner) runE(cmd *cobra.Command, args []string) error {
	var (
		reader *bufio.Reader
		ctx    = registry.New()
		err    error
	)
	if reader, err = flags.OpenFile(args[0]); err != nil {
		return err
	}
	p := Parser{
		reader:  csv.NewReader(utfbom.SkipOnly(reader)),
		journal: journal.New(ctx),
	}
	if p.account, err = r.accountFlag.Value(ctx.Accounts()); err != nil {
		return err
	}
	if err = p.parse(); err != nil {
		return err
	}
	out := bufio.NewWriter(cmd.OutOrStdout())
	defer out.Flush()
	return journal.Print(out, p.journal)
}

func init() {
	importer.Register(CreateCmd)
}

// Parser is a parser for account statements
type Parser struct {
	reader  *csv.Reader
	account *model.Account
	journal *journal.Journal

	currency *model.Commodity
}

func (p *Parser) parse() error {
	p.reader.LazyQuotes = true
	p.reader.TrimLeadingSpace = true
	p.reader.Comma = ';'
	p.reader.FieldsPerRecord = -1

	if _, err := p.readHeader("Buchungsart:"); err != nil {
		return err
	}
	if _, err := p.readHeader("Konto:"); err != nil {
		return err
	}
	if s, err := p.readHeader("Währung:"); err != nil {
		return err
	} else {
		sym := strings.Trim(s, "=\"")
		if p.currency, err = p.journal.Registry.GetCommodity(sym); err != nil {
			return err
		}
	}
	if err := p.readHeaders(); err != nil {
		return err
	}
	for {
		ok, err := p.readBookingLine()
		if err != nil {
			return err
		}
		if !ok {
			break
		}
	}
	for {
		err := p.readDisclaimer()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

func (p *Parser) readHeader(header ...string) (string, error) {
	rec, err := p.reader.Read()
	if err != nil {
		return "", err
	}
	if !slices.Contains(header, rec[0]) {
		return "", fmt.Errorf("got %q, want one of %#v", rec[0], header)
	}
	return rec[1], nil
}

func (p *Parser) readHeaders() error {
	_, err := p.reader.Read()
	return err
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

func (p *Parser) readBookingLine() (bool, error) {
	rec, err := p.reader.Read()
	if err != nil {
		return false, err
	}
	if len(rec) < 5 || len(rec) > 6 {
		return false, nil
	}
	date, err := time.Parse("02.01.2006", rec[bfBuchungsdatum])
	if err != nil {
		return false, err
	}
	amount, err := parseAmount(rec[bfGutschriftInCHF], rec[bfLastschriftInCHF])
	if err != nil {
		return false, err
	}
	p.journal.AddTransaction(transaction.Builder{
		Date:        date,
		Description: strings.TrimSpace(rec[bfAvisierungstext]),
		Postings: posting.Builder{
			Credit:    p.journal.Registry.TBDAccount(),
			Debit:     p.account,
			Commodity: p.currency,
			Amount:    amount,
		}.Build(),
	}.Build())
	return true, nil
}

func (p *Parser) readDisclaimer() error {
	rec, err := p.reader.Read()
	if err != nil {
		return err
	}
	if len(rec) != 1 {
		return fmt.Errorf("expected one field, got %#v", rec)
	}
	return err
}

func parseAmount(gutschrift, lastschrift string) (decimal.Decimal, error) {
	switch {
	case len(gutschrift) > 0 && len(lastschrift) == 0:
		return decimal.NewFromString(gutschrift)
	case len(gutschrift) == 0 && len(lastschrift) > 0:
		return decimal.NewFromString(lastschrift)
	default:
		return decimal.Zero, fmt.Errorf("invalid amount fields %q %q", gutschrift, lastschrift)
	}
}
