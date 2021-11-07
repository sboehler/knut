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

	"github.com/sboehler/knut/cmd/flags"
	"github.com/sboehler/knut/cmd/importer"
	"github.com/sboehler/knut/lib/ledger"
	"github.com/sboehler/knut/lib/model/accounts"
	"github.com/sboehler/knut/lib/model/commodities"
	"github.com/sboehler/knut/lib/printer"
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
	account, err := flags.GetAccountFlag(cmd, "account")
	if err != nil {
		return err
	}
	f, err := os.Open(args[0])
	if err != nil {
		return err
	}
	var (
		reader = csv.NewReader(bufio.NewReader(f))
		p      = parser{
			reader:  reader,
			account: account,
			builder: ledger.NewBuilder(nil, nil),
		}
	)
	if err = p.parse(); err != nil {
		return err
	}
	var w = bufio.NewWriter(cmd.OutOrStdout())
	defer w.Flush()
	_, err = printer.New().PrintLedger(w, p.builder.Build())
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
	p.reader.FieldsPerRecord = 0

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

type bookingField int

const (
	bfCompletedDate bookingField = iota
	bfReference
	bfPaidOut
	bfPaidIn
	bfExchangeOut
	bfExchangeIn
	bfBalance
	bfExchangeRate
	bfCategory
)

var re = regexp.MustCompile(`Paid Out \(([A-Za-z]+)\)`)

func (p *parser) parseHeader(r []string) error {
	if len(r) != 9 {
		return fmt.Errorf("expected record with 9 items, got %v", r)
	}
	var groups = re.FindStringSubmatch(r[bfPaidOut])
	if len(groups) != 2 {
		return fmt.Errorf("could not extract currency from header field: %q", r[bfPaidOut])
	}
	var err error
	p.currency, err = commodities.Get(groups[1])
	return err
}

var (
	fxSellRegex = regexp.MustCompile(`Sold [A-Z]+ to [A-Z]+`)
	fxBuyRegex  = regexp.MustCompile(`Bought [A-Z]+ from [A-Z]+`)
	space       = regexp.MustCompile(`\s+`)
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
		balance, err := parseDecimal(r[6])
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
	for _, field := range []bookingField{bfReference, bfExchangeRate, bfCategory} {
		words = append(words, r[field])
	}
	var (
		desc   = strings.TrimSpace(space.ReplaceAllString(strings.Join(words, " "), " "))
		amount decimal.Decimal
		field  bookingField
		sign   = decimal.NewFromInt(1)
	)
	switch {

	case len(r[bfPaidOut]) > 0 && len(r[bfPaidIn]) == 0:
		field = bfPaidOut
		sign = sign.Neg()
	case len(r[bfPaidOut]) == 0 && len(r[bfPaidIn]) > 0:
		field = bfPaidIn
	default:
		return fmt.Errorf("invalid record with two amounts: %v", r)
	}
	if amount, err = parseDecimal(r[field]); err != nil {
		return err
	}
	amount = amount.Mul(sign)
	var t = ledger.Transaction{
		Date:        date,
		Description: desc,
	}
	switch {
	case fxSellRegex.MatchString(r[bfReference]):
		otherCommodity, otherAmount, err := parseCombiField(r[bfExchangeOut])
		if err != nil {
			return err
		}
		t.Postings = []ledger.Posting{
			ledger.NewPosting(accounts.ValuationAccount(), p.account, p.currency, amount),
			ledger.NewPosting(accounts.ValuationAccount(), p.account, otherCommodity, otherAmount),
		}
	case fxBuyRegex.MatchString(r[bfReference]):
		otherCommodity, otherAmount, err := parseCombiField(r[bfExchangeIn])
		if err != nil {
			return err
		}
		t.Postings = []ledger.Posting{
			ledger.NewPosting(accounts.ValuationAccount(), p.account, p.currency, amount),
			ledger.NewPosting(accounts.ValuationAccount(), p.account, otherCommodity, otherAmount.Neg()),
		}
	default:
		t.Postings = []ledger.Posting{
			ledger.NewPosting(accounts.TBDAccount(), p.account, p.currency, amount),
		}
	}
	p.builder.AddTransaction(&t)
	return nil
}

func parseCombiField(f string) (*commodities.Commodity, decimal.Decimal, error) {
	var fs = strings.Fields(f)
	if len(fs) != 2 {
		return nil, decimal.Decimal{}, fmt.Errorf("expected currency and amount, got %s", f)
	}
	var (
		otherCommodity *commodities.Commodity
		otherAmount    decimal.Decimal
		err            error
	)
	if otherCommodity, err = commodities.Get(fs[0]); err != nil {
		return nil, decimal.Decimal{}, err
	}
	if otherAmount, err = parseDecimal(fs[1]); err != nil {
		return nil, decimal.Decimal{}, err
	}
	return otherCommodity, otherAmount, nil
}

func parseDecimal(s string) (decimal.Decimal, error) {
	s = strings.ReplaceAll(s, "'", "")
	return decimal.NewFromString(s)
}
