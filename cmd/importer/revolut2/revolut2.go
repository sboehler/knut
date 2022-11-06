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

package revolut2

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"time"

	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"

	"github.com/sboehler/knut/cmd/flags"
	"github.com/sboehler/knut/cmd/importer"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
	"github.com/sboehler/knut/lib/journal/ast/printer"
)

// CreateCmd creates the command.
func CreateCmd() *cobra.Command {
	var r runner
	cmd := &cobra.Command{
		Use:   "revolut2",
		Short: "Import Revolut CSV account statements",
		Long:  `Download one CSV file per account through their app. Make sure the app language is set to English, as they use localized formats.`,

		RunE: r.run,
	}
	r.setupFlags(cmd)
	return cmd
}

func init() {
	importer.Register(CreateCmd)
}

type runner struct {
	account, feeAccount flags.AccountFlag
}

func (r *runner) setupFlags(cmd *cobra.Command) {
	cmd.Flags().VarP(&r.account, "account", "a", "account name")
	cmd.Flags().VarP(&r.feeAccount, "fee", "f", "fee account name")
	cmd.MarkFlagRequired("account")
	cmd.MarkFlagRequired("fee")
}

func (r *runner) run(cmd *cobra.Command, args []string) error {
	var (
		ctx = journal.NewContext()
		f   *bufio.Reader
		err error
	)
	a := ast.New(ctx)
	for _, path := range args {
		if f, err = flags.OpenFile(path); err != nil {
			return err
		}
		p := parser{
			reader: csv.NewReader(f),
			ast:    a,
		}
		if p.account, err = r.account.Value(ctx); err != nil {
			return err
		}
		if p.feeAccount, err = r.feeAccount.Value(ctx); err != nil {
			return err
		}
		if err = p.parse(); err != nil {
			return err
		}
	}
	out := bufio.NewWriter(cmd.OutOrStdout())
	defer out.Flush()
	_, err = printer.New().PrintLedger(out, a.SortedDays())
	return err
}

type parser struct {
	reader              *csv.Reader
	account, feeAccount *journal.Account
	ast                 *ast.Journal
	balance             journal.Amounts
}

func (p *parser) parse() error {
	p.reader.TrimLeadingSpace = true
	p.reader.Comma = ','
	p.reader.FieldsPerRecord = 10
	p.balance = make(journal.Amounts)

	if err := p.parseHeader(); err != nil {
		return err
	}
	for {
		if err := p.parseBooking(); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}
	p.addBalances()
	return nil
}

type bookingField int

const (
	bfType bookingField = iota
	bfProduct
	bfStartedDate
	bfCompletedDate
	bfDescription
	bfAmount
	bfFee
	bfCurrency
	bfState
	bfBalance
)

func (p *parser) parseHeader() error {
	r, err := p.reader.Read()
	if err != nil {
		return err
	}
	header := []string{"Type", "Product", "Started Date", "Completed Date", "Description", "Amount", "Fee", "Currency", "State", "Balance"}
	for i := range r {
		if r[i] != header[i] {
			return fmt.Errorf("invalid header: %v", r)
		}
	}
	return nil
}

func (p *parser) parseBooking() error {
	r, err := p.reader.Read()
	if err != nil {
		return err
	}
	if r[bfCompletedDate] == "" {
		return nil
	}
	d, err := time.Parse("2006-01-02", r[bfCompletedDate][:10])
	if err != nil {
		return fmt.Errorf("invalid started date in row %v: %w", r, err)
	}
	c, err := p.ast.Context.GetCommodity(r[bfCurrency])
	if err != nil {
		return fmt.Errorf("invalid commodity in row %v: %v", r, err)
	}
	amt, err := decimal.NewFromString(r[bfAmount])
	if err != nil {
		return fmt.Errorf("invalid amount in row %v: %v", r, err)
	}
	t := ast.TransactionBuilder{
		Date:        d,
		Description: r[bfDescription],
		Postings: []ast.Posting{
			ast.NewPosting(p.ast.Context.TBDAccount(), p.account, c, amt),
		},
	}
	fee, err := decimal.NewFromString(r[bfFee])
	if err != nil {
		return fmt.Errorf("invalid fee in row %v: %v", r, err)
	}
	if !fee.IsZero() {
		t.Postings = append(t.Postings, ast.NewPosting(p.account, p.feeAccount, c, fee))
	}
	p.ast.AddTransaction(t.Build())
	bal, err := decimal.NewFromString(r[bfBalance])
	if err != nil {
		return fmt.Errorf("invalid balance in row %v: %v", r, err)
	}
	p.balance[journal.DateCommodityKey(d, c)] = bal
	return nil
}

func (p *parser) addBalances() {
	for k, bal := range p.balance {
		p.ast.AddAssertion(&ast.Assertion{
			Date:      k.Date,
			Commodity: k.Commodity,
			Amount:    bal,
			Account:   p.account,
		})
	}
}
