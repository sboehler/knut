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
		Use:   "ch.swisscard",
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
	account       flags.AccountFlag
	addCardTag    bool
	cardTagPrefix string
}

func (r *runner) setupFlags(cmd *cobra.Command) {
	cmd.Flags().VarP(&r.account, "account", "a", "account name")
	cmd.Flags().BoolVar(&r.addCardTag, "add-card-tag", false, "add card number to every transaction as a tag")
	cmd.Flags().StringVar(&r.cardTagPrefix, "card-tag-prefix", "card:", "prefix of the card number tag")
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
		reader:        csv.NewReader(f),
		builder:       journal.New(ctx),
		addCardTag:    r.addCardTag,
		cardTagPrefix: r.cardTagPrefix,
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
	reader        *csv.Reader
	account       *journal.Account
	builder       *journal.Journal
	addCardTag    bool
	cardTagPrefix string
}

func (p *parser) parse() error {
	p.reader.TrimLeadingSpace = true
	p.reader.FieldsPerRecord = len(fieldHeaders)
	firstLine := true // First line contains field names so it requires special treatment.
	for {
		err := p.readLine(firstLine)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		firstLine = false
	}
}

var fieldHeaders = []string{
	"Transaction date",
	"Description",
	"Card number",
	"Currency",
	"Amount",
	"Debit/Credit",
	"Status",
	"Category",
}

func (p *parser) readLine(firstLine bool) error {
	r, err := p.reader.Read()
	if err != nil {
		return err
	}
	if !firstLine {
		return p.parseBooking(r)
	}
	for i, expected := range fieldHeaders {
		if r[i] != expected {
			return fmt.Errorf("unexpected CSV header (column %d): wanted %q, got %q", i+1, expected, r[i])
		}
	}
	return nil
}

func (p *parser) parseBooking(r []string) error {
	date, err := time.Parse("02.01.2006", r[0])
	if err != nil {
		return err
	}

	desc := strings.TrimSpace(r[1])

	tags := []journal.Tag{}
	cardNo := strings.ReplaceAll(r[2], " ", "")
	if p.addCardTag && cardNo != "" {
		tags = append(tags, journal.Tag(p.cardTagPrefix+cardNo))
	}

	commodity, err := p.builder.Context.GetCommodity(r[3])
	if err != nil {
		return err
	}

	amount, err := decimal.NewFromString(r[4])
	if err != nil {
		return err
	}

	if (r[5] == "Debit" && !amount.IsPositive()) || (r[5] == "Credit" && !amount.IsNegative()) {
		return fmt.Errorf("%s transaction should be %s, not %s", r[5], amount.Neg(), amount)
	}

	// r[6] ("Status") field is ignored

	category := strings.TrimSpace(r[7])
	if category != "" {
		desc += " / " + category
	}

	p.builder.AddTransaction(journal.TransactionBuilder{
		Date:        date,
		Description: desc,
		Postings: journal.PostingBuilder{
			Credit:    p.account,
			Debit:     p.builder.Context.TBDAccount(),
			Amount:    amount,
			Commodity: commodity,
		}.Build(),
		Tags: tags,
	}.Build())

	return nil
}
