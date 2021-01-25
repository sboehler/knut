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

package postfinance

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"
	"golang.org/x/text/encoding/charmap"

	"github.com/sboehler/knut/cmd/importer"
	"github.com/sboehler/knut/lib/ledger"
	"github.com/sboehler/knut/lib/model/accounts"
	"github.com/sboehler/knut/lib/model/commodities"
	"github.com/sboehler/knut/lib/printer"
	"github.com/sboehler/knut/lib/scanner"
)

// CreateCmd creates the cobra command.
func CreateCmd() *cobra.Command {

	var cmd = cobra.Command{
		Use:   "ch.postfinance",
		Short: "Import Postfinance CSV account statements",

		Args: cobra.ExactValidArgs(1),

		RunE: run,
	}
	cmd.Flags().StringP("account", "a", "", "account name")
	return &cmd
}

func run(cmd *cobra.Command, args []string) error {
	accountName, err := cmd.Flags().GetString("account")
	if err != nil {
		return err
	}
	accountScanner, err := scanner.New(strings.NewReader(accountName), "")
	if err != nil {
		return err
	}
	account, err := scanner.ParseAccount(accountScanner)
	if err != nil {
		return err
	}
	f, err := os.Open(args[0])
	if err != nil {
		return err
	}
	scanner, err := scanner.New(bufio.NewReader(charmap.ISO8859_1.NewDecoder().Reader(f)), args[0])
	if err != nil {
		return err
	}
	var p = Parser{scanner, account, ledger.NewBuilder(ledger.Filter{})}
	if err = p.parse(); err != nil {
		return err
	}
	var w = bufio.NewWriter(cmd.OutOrStdout())
	defer w.Flush()
	_, err = printer.PrintLedger(w, p.builder.Build())
	return err
}

func init() {
	importer.Register(CreateCmd)
}

// Parser is a parser for account statements
type Parser struct {
	*scanner.Scanner
	account *accounts.Account
	builder *ledger.Builder
}

func (p *Parser) parse() error {
	if p.Current() == 'D' {
		if err := p.consumeStringField("Datum von:"); err != nil {
			return err
		}
		if err := p.ignoreField(); err != nil {
			return err
		}
	}
	for _, s := range [][]string{
		{"Buchungsart:", "Entry type:"},
		{"Konto:", "Account:"},
	} {
		if err := p.consumeStringField(s...); err != nil {
			return err
		}
		if err := p.ignoreField(); err != nil {
			return err
		}
	}

	if err := p.consumeStringField("Währung:", "Currency:"); err != nil {
		return err
	}

	s, err := p.getField()
	if err != nil {
		return err
	}
	var commodity = commodities.Get(s)

	// ignore 6 header fields
	for i := 0; i < 6; i++ {
		if err := p.ignoreField(); err != nil {
			return err
		}
	}

	for !unicode.IsControl(p.Current()) {

		s, err := p.getField()
		if err != nil {
			return err
		}
		d, err := time.Parse("2006-01-02", s)
		if err != nil {
			return err
		}
		description, err := p.getField()
		if err != nil {
			return err
		}
		if p.Current() == ';' {
			if err := p.consumeDelimiter(); err != nil {
				return err
			}
			s, err = p.getField()
			if err != nil {
				return err
			}
		} else {
			s, err = p.getField()
			if err != nil {
				return err
			}
			if err := p.consumeDelimiter(); err != nil {
				return err
			}
		}
		amt, err := decimal.NewFromString(s)
		if err != nil {
			return err
		}

		for i := 0; i < 2; i++ {
			if err := p.ignoreField(); err != nil {
				return err
			}
		}
		var drAccount, crAccount *accounts.Account
		if amt.IsPositive() {
			crAccount, drAccount = accounts.TBDAccount(), p.account
		} else {
			crAccount, drAccount = p.account, accounts.TBDAccount()
		}

		var (
			postings = []*ledger.Posting{
				{
					Amount:    amt.Abs(),
					Credit:    crAccount,
					Debit:     drAccount,
					Commodity: commodity,
				},
			}
			t = &ledger.Transaction{
				Date:        d,
				Description: description,
				Postings:    postings,
			}
		)
		p.builder.AddTransaction(t)

	}

	for i := 0; i < 3; i++ {
		if err := p.ignoreField(); err != nil {
			return err
		}
	}
	if p.Current() != scanner.EOF {
		return fmt.Errorf("Expected EOF, got %b", p.Current())
	}
	return nil
}

func (p *Parser) consumeStringField(ss ...string) error {
	s, err := p.getField()
	if err != nil {
		return err
	}
	if err := oneOf(s, ss...); err != nil {
		return err
	}
	return nil
}

func (p *Parser) ignoreField() error {
	_, err := p.getField()
	return err
}

func (p *Parser) getField() (string, error) {
	var (
		err error
		s   string
	)
	if p.Current() == '"' {
		if s, err = scanner.ReadQuotedString(p.Scanner); err != nil {
			return s, err
		}
	} else if p.Current() != '\n' && p.Current() != '\r' && p.Current() != ';' {
		if s, err = p.readUnquotedField(); err != nil {
			return s, err
		}
	}
	if err := p.consumeDelimiter(); err != nil {
		return s, err
	}
	return s, nil
}

func (p *Parser) consumeDelimiter() error {
	if p.Current() == ';' {
		err := p.ConsumeRune(';')
		return err
	}
	if p.Current() == '\n' {
		err := p.ConsumeRune('\n')
		return err
	}
	if p.Current() == '\r' {
		if err := p.ConsumeRune('\r'); err != nil {
			return err
		}
		if err := p.ConsumeRune('\n'); err != nil {
			return err
		}
		return nil
	}
	if p.Current() == scanner.EOF {
		return nil
	}
	return fmt.Errorf("Expected delimiter, newline or EOF, got %c", p.Current())
}

func (p *Parser) readUnquotedField() (string, error) {
	return p.ReadWhile(func(r rune) bool {
		return r != ';' && r != '\n' && r != '\r' && r != scanner.EOF
	})
}

func oneOf(s string, ss ...string) error {
	if len(ss) == 0 {
		return nil
	}
	for _, c := range ss {
		if s == c {
			return nil
		}
	}
	return fmt.Errorf("Expected %q, got %q", ss, s)
}
