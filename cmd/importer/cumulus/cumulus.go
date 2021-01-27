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

	"github.com/sboehler/knut/cmd/importer"
	"github.com/sboehler/knut/lib/ledger"
	"github.com/sboehler/knut/lib/model/accounts"
	"github.com/sboehler/knut/lib/model/commodities"
	"github.com/sboehler/knut/lib/printer"
	"github.com/sboehler/knut/lib/scanner"
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
	accountName, err := cmd.Flags().GetString("account")
	if err != nil {
		return err
	}
	s, err := scanner.New(strings.NewReader(accountName), "")
	if err != nil {
		return err
	}
	account, err := scanner.ParseAccount(s)
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
	if err != nil {
		return err
	}
	if ok, err := p.parseRounding(r); ok || err != nil {
		p.last = nil
		return err
	}
	if ok, err := p.parseFXComment(r); ok || err != nil {
		return err
	}
	if ok, err := p.parseBooking(r); ok || err != nil {
		return err
	}
	p.last = nil
	return nil
}

var dateRegex = regexp.MustCompile(`\d\d.\d\d.\d\d\d\d`)

func (p *parser) parseBooking(r []string) (bool, error) {
	if !dateRegex.MatchString(r[0]) || !dateRegex.MatchString(r[1]) {
		return false, nil
	}
	if len(r) != 5 {
		return false, fmt.Errorf("expected five items, got %v", r)
	}
	var (
		err                error
		desc               = r[2]
		crAmount, drAmount = r[4], r[3]
		amt                decimal.Decimal
		d                  time.Time
	)
	if d, err = time.Parse("02.01.2006", r[0]); err != nil {
		return false, err
	}
	switch {
	// credit booking
	case len(crAmount) > 0 && len(drAmount) == 0:
		amt, err = decimal.NewFromString(crAmount)
		if err != nil {
			return false, err
		}
	// debit booking
	case len(crAmount) == 0 && len(drAmount) > 0:
		amt, err = decimal.NewFromString(drAmount)
		if err != nil {
			return false, err
		}
	default:
		return false, fmt.Errorf("row has invalid amounts: %v", r)
	}
	p.last = &ledger.Transaction{
		Date:        d,
		Description: desc,
		Postings: []*ledger.Posting{
			ledger.NewPosting(p.account, accounts.TBDAccount(), commodities.Get("CHF"), amt),
		},
	}
	p.builder.AddTransaction(p.last)
	return true, nil

}

func (p *parser) parseFXComment(r []string) (bool, error) {
	if len(r) != 5 || len(r[0]) > 0 || len(r[1]) > 0 || len(r[2]) == 0 || len(r[3]) > 0 || len(r[4]) > 0 {
		return false, nil
	}
	if p.last == nil {
		return false, fmt.Errorf("fx comment but no previous transaction")
	}
	p.last.Description = fmt.Sprintf("%s %s", p.last.Description, r[2])
	return true, nil
}

func (p *parser) parseRounding(r []string) (bool, error) {
	if !dateRegex.MatchString(r[0]) || r[1] != "Rundungskorrektur" {
		return false, nil
	}
	if len(r) != 4 {
		return false, fmt.Errorf("expected three items, got %v", r)
	}
	var (
		err                error
		desc               = r[1]
		crAmount, drAmount = r[3], r[2]
		amt                decimal.Decimal
		d                  time.Time
	)
	if d, err = time.Parse("02.01.2006", r[0]); err != nil {
		return false, err
	}
	switch {
	// credit booking
	case len(crAmount) > 0 && len(drAmount) == 0:
		amt, err = decimal.NewFromString(crAmount)
		if err != nil {
			return false, err
		}
		amt = amt.Neg()
	// debit booking
	case len(crAmount) == 0 && len(drAmount) > 0:
		amt, err = decimal.NewFromString(drAmount)
		if err != nil {
			return false, err
		}
	default:
		return false, fmt.Errorf("row has invalid amounts: %v", r)
	}
	p.builder.AddTransaction(&ledger.Transaction{
		Date:        d,
		Description: desc,
		Postings: []*ledger.Posting{
			ledger.NewPosting(p.account, accounts.TBDAccount(), commodities.Get("CHF"), amt),
		},
	})
	return true, nil
}
