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

package revolut

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

	"github.com/sboehler/knut/cmd/flags"
	"github.com/sboehler/knut/cmd/importer"
	"github.com/sboehler/knut/lib/journal"
)

// CreateCmd creates the command.
func CreateCmd() *cobra.Command {
	var r runner
	cmd := &cobra.Command{
		Use:   "revolut",
		Short: "Import Revolut CSV account statements",
		Long:  `Download one CSV file per account through their app. Make sure the app language is set to English, as they use localized formats.`,

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
	cmd.MarkFlagRequired("account")
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
	p := parser{
		reader: csv.NewReader(f),
		ast:    journal.New(ctx),
	}
	if p.account, err = r.account.Value(ctx); err != nil {
		return err
	}
	if err = p.parse(); err != nil {
		return err
	}
	out := bufio.NewWriter(cmd.OutOrStdout())
	defer out.Flush()
	_, err = journal.NewPrinter().PrintLedger(out, p.ast.SortedDays())
	return err
}

type parser struct {
	reader   *csv.Reader
	account  *journal.Account
	ast      *journal.Journal
	currency *journal.Commodity
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
	p.currency, err = p.ast.Context.GetCommodity(groups[1])
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
		p.ast.AddAssertion(&journal.Assertion{
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
	var t = journal.TransactionBuilder{
		Date:        date,
		Description: desc,
	}
	switch {
	case fxSellRegex.MatchString(r[bfReference]):
		otherCommodity, otherAmount, err := p.parseCombiField(r[bfExchangeOut])
		if err != nil {
			return err
		}
		t.Postings = []*journal.Posting{
			journal.PostingBuilder{
				Credit:    p.ast.Context.ValuationAccount(),
				Debit:     p.account,
				Commodity: p.currency,
				Amount:    amount,
			}.Build(),
			journal.PostingBuilder{
				Credit:    p.ast.Context.ValuationAccount(),
				Debit:     p.account,
				Commodity: otherCommodity,
				Amount:    otherAmount,
			}.Build(),
		}
	case fxBuyRegex.MatchString(r[bfReference]):
		otherCommodity, otherAmount, err := p.parseCombiField(r[bfExchangeIn])
		if err != nil {
			return err
		}
		t.Postings = []*journal.Posting{
			journal.PostingBuilder{
				Credit:    p.ast.Context.ValuationAccount(),
				Debit:     p.account,
				Commodity: p.currency,
				Amount:    amount,
			}.Build(),
			journal.PostingBuilder{
				Credit:    p.ast.Context.ValuationAccount(),
				Debit:     p.account,
				Commodity: otherCommodity,
				Amount:    otherAmount.Neg(),
			}.Build(),
		}
	default:
		t.Postings = []*journal.Posting{
			journal.PostingBuilder{
				Credit:    p.ast.Context.TBDAccount(),
				Debit:     p.account,
				Commodity: p.currency,
				Amount:    amount,
			}.Build(),
		}
	}
	p.ast.AddTransaction(t.Build())
	return nil
}

func (p *parser) parseCombiField(f string) (*journal.Commodity, decimal.Decimal, error) {
	var fs = strings.Fields(f)
	if len(fs) != 2 {
		return nil, decimal.Decimal{}, fmt.Errorf("expected currency and amount, got %s", f)
	}
	var (
		otherCommodity *journal.Commodity
		otherAmount    decimal.Decimal
		err            error
	)
	if otherCommodity, err = p.ast.Context.GetCommodity(fs[0]); err != nil {
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
