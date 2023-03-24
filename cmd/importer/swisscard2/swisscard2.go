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

package swisscard

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
)

// CreateCmd creates the command.
func CreateCmd() *cobra.Command {
	var r runner
	cmd := &cobra.Command{
		Use:   "ch.swisscard2",
		Short: "Import Swisscard credit card statements",
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
		reader:  csv.NewReader(f),
		builder: journal.New(ctx),
	}
	if p.account, err = r.account.Value(ctx); err != nil {
		return err
	}
	if err = p.parse(); err != nil {
		return err
	}
	w := bufio.NewWriter(cmd.OutOrStdout())
	defer w.Flush()
	_, err = journal.NewPrinter().PrintLedger(w, p.builder.ToLedger())
	return err
}

type parser struct {
	reader  *csv.Reader
	account *journal.Account
	builder *journal.Journal
}

func (p *parser) parse() error {
	p.reader.TrimLeadingSpace = true
	if err := p.readHeader(); err != nil {
		return err
	}
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

type column int

const (
	transaktionsdatum column = iota
	beschreibung
	kartennummer
	währung
	betrag
	debitKredit
	status
	kategorie
)

func (p *parser) readHeader() error {
	_, err := p.reader.Read()
	return err
}

func (p *parser) readLine() error {
	r, err := p.reader.Read()
	if err != nil {
		return err
	}
	return p.parseBooking(r)
}

func (p *parser) parseBooking(r []string) error {
	d, err := time.Parse("02.01.2006", r[transaktionsdatum])
	if err != nil {
		return fmt.Errorf("invalid date in record %v: %w", r, err)
	}
	c := p.builder.Context.Commodity(r[währung])
	amt, err := decimal.NewFromString(r[betrag])
	if err != nil {
		return fmt.Errorf("invalid amount in record %v: %w", r, err)
	}
	p.builder.AddTransaction(journal.TransactionBuilder{
		Date:        d,
		Description: fmt.Sprintf("%s %s", r[beschreibung], r[kartennummer]),
		Postings: journal.PostingBuilder{
			Credit:    p.account,
			Debit:     p.builder.Context.TBDAccount(),
			Commodity: c,
			Amount:    amt,
		}.Build(),
	}.Build())
	return nil
}
