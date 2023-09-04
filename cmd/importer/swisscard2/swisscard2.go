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
	"github.com/sboehler/knut/lib/model"
	"github.com/sboehler/knut/lib/model/posting"
	"github.com/sboehler/knut/lib/model/registry"
	"github.com/sboehler/knut/lib/model/transaction"
)

// CreateCmd creates the command.
func CreateCmd() *cobra.Command {
	var r runner
	cmd := &cobra.Command{
		Use:   "ch.swisscard2",
		Short: "Import Swisscard credit card statements (from mid 2023)",
		Long:  `Download the CSV file from their account management tool.`,

		Args: cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),

		RunE: r.run,
	}
	r.setupFlags(cmd)
	return cmd
}

func init() {
	importer.RegisterImporter(CreateCmd)
}

type runner struct {
	account flags.AccountFlag
}

func (r *runner) setupFlags(cmd *cobra.Command) {
	cmd.Flags().VarP(&r.account, "account", "a", "account name")
	cmd.MarkFlagRequired("account")

}

func (r *runner) run(cmd *cobra.Command, args []string) error {
	ctx := registry.New()
	f, err := flags.OpenFile(args[0])
	if err != nil {
		return err
	}
	account, err := r.account.Value(ctx.Accounts())
	if err != nil {
		return err
	}
	p := parser{
		reader:  csv.NewReader(f),
		journal: journal.New(ctx),
		account: account,
	}
	if err = p.parse(); err != nil {
		return err
	}
	w := bufio.NewWriter(cmd.OutOrStdout())
	defer w.Flush()
	return journal.Print(w, p.journal)
}

type parser struct {
	reader  *csv.Reader
	account *model.Account
	journal *journal.Journal
}

func (p *parser) parse() error {
	p.reader.TrimLeadingSpace = true
	p.reader.FieldsPerRecord = 8

	if err := p.readHeader(); err != nil {
		return err
	}
	for {
		err := p.readBooking()
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

func (p *parser) readBooking() error {
	r, err := p.reader.Read()
	if err != nil {
		return err
	}
	d, err := time.Parse("02.01.2006", r[transaktionsdatum])
	if err != nil {
		return fmt.Errorf("invalid date in record %v: %w", r, err)
	}
	c := p.journal.Registry.Commodities().MustGet(r[währung])
	quantity, err := decimal.NewFromString(r[betrag])
	if err != nil {
		return fmt.Errorf("invalid amount in record %v: %w", r, err)
	}
	p.journal.AddTransaction(transaction.Builder{
		Date:        d,
		Description: fmt.Sprintf("%s / %s / %s / %s", r[beschreibung], r[kartennummer], r[kategorie], r[debitKredit]),
		Postings: posting.Builder{
			Credit:    p.account,
			Debit:     p.journal.Registry.Accounts().TBDAccount(),
			Commodity: c,
			Quantity:  quantity,
		}.Build(),
	}.Build())
	return nil
}
