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

package revolut

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
	cmd := cobra.Command{
		Use:   "revolut",
		Short: "Import Revolut CSV account statements",
		Long:  `Download one CSV file per account through their app. Make sure the app language is set to English, as they use localized formats.`,

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
		builder: ledger.NewBuilder(ledger.Options{}),
	}
	if err = p.parse(); err != nil {
		return err
	}
	w := bufio.NewWriter(cmd.OutOrStdout())
	defer w.Flush()
	_, err = printer.Printer{}.PrintLedger(w, p.builder.Build())
	return err
}

type parser struct {
	reader   *csv.Reader
	account  *accounts.Account
	builder  *ledger.Builder
	currency *commodities.Commodity
	date     time.Time
}

func (p *parser) parse() error {
	p.reader.TrimLeadingSpace = true
	p.reader.Comma = ';'

	var (
		r   []string
		err error
	)
	if r, err = p.reader.Read(); err != nil {
		return err
	}
	if err = p.parseHeader(r); err != nil {
		return err
	}
	for {
		if r, err = p.reader.Read(); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if err = p.parseBooking(r); err != nil {
			return err
		}
	}
}

var re = regexp.MustCompile(`Paid Out \(([A-Za-z]+)\)`)

func (p *parser) parseHeader(r []string) error {
	if len(r) != 9 {
		return fmt.Errorf("expected record with 9 items, got %v", r)
	}
	groups := re.FindStringSubmatch(r[2])
	if len(groups) != 2 {
		return fmt.Errorf("could not extract currency from header field: %q", r[2])
	}
	p.currency = commodities.Get(groups[1])
	return nil
}

var (
	dateRegex = regexp.MustCompile(`\d\d.\d\d.\d\d\d\d`)
	replacer  = strings.NewReplacer("'", "")
)

var (
	fxSellRegex = regexp.MustCompile(`Sold [A-Z]+ to [A-Z]+`)
	fxBuyRegex  = regexp.MustCompile(`Bought [A-Z]+ from [A-Z]+`)
)

func (p *parser) parseBooking(r []string) error {
	if len(r) != 9 {
		return fmt.Errorf("expected record with 9 items, got %v", r)
	}
	date, err := time.Parse("2 Jan 2006", r[0])
	if err != nil {
		return err
	}
	if date != p.date {
		balance, err := decimal.NewFromString(replacer.Replace(r[6]))
		if err != nil {
			return err
		}
		p.builder.AddAssertion(&ledger.Assertion{
			Date:      date,
			Account:   p.account,
			Amount:    balance,
			Commodity: p.currency,
		})
		p.date = date
	}

	var words []string
	for _, i := range []int{1, 7, 8} {
		words = append(words, strings.Fields(r[i])...)
	}
	desc := strings.Join(words, " ")
	var amt decimal.Decimal
	if len(r[2]) > 0 && len(r[3]) == 0 {
		// credit booking
		if amt, err = decimal.NewFromString(replacer.Replace(r[2])); err != nil {
			return err
		}
	} else if len(r[2]) == 0 && len(r[3]) > 0 {
		// debit booking
		if amt, err = decimal.NewFromString(replacer.Replace(r[3])); err != nil {
			return err
		}
		amt = amt.Neg()
	} else {
		return fmt.Errorf("invalid record with two amounts: %v", r)
	}

	var t = ledger.Transaction{
		Date:        date,
		Description: desc,
	}
	switch {
	case fxSellRegex.MatchString(r[1]):
		otherCommodity, otherAmount, err := parseCombiField(r[4])
		if err != nil {
			return err
		}
		t.Postings = []*ledger.Posting{
			ledger.NewPosting(p.account, accounts.ValuationAccount(), p.currency, amt),
			ledger.NewPosting(accounts.ValuationAccount(), p.account, otherCommodity, otherAmount),
		}
	case fxBuyRegex.MatchString(r[1]):
		otherCommodity, otherAmount, err := parseCombiField(r[5])
		if err != nil {
			return err
		}
		t.Postings = []*ledger.Posting{
			ledger.NewPosting(p.account, accounts.ValuationAccount(), p.currency, amt),
			ledger.NewPosting(accounts.ValuationAccount(), p.account, otherCommodity, otherAmount.Neg()),
		}
	default:
		t.Postings = []*ledger.Posting{
			ledger.NewPosting(p.account, accounts.TBDAccount(), p.currency, amt),
		}
	}
	p.builder.AddTransaction(&t)
	return nil
}

func parseCombiField(f string) (*commodities.Commodity, decimal.Decimal, error) {
	fs := strings.Fields(f)
	if len(fs) != 2 {
		return nil, decimal.Decimal{}, fmt.Errorf("expected currency and amount, got %s", f)
	}
	otherCommodity := commodities.Get(fs[0])
	otherAmount, err := decimal.NewFromString(replacer.Replace(fs[1]))
	if err != nil {
		return nil, decimal.Decimal{}, err
	}
	return otherCommodity, otherAmount, nil
}
