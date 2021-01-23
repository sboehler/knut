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

package swisscard

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
		Use:   "ch.swisscard",
		Short: "Import Swisscard credit card statements",
		Long:  `Download the CSV file from their account management tool.`,

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
	var (
		reader = csv.NewReader(bufio.NewReader(f))
		p      = parser{
			reader:  reader,
			account: account,
			builder: ledger.NewBuilder(ledger.Filter{}),
		}
	)
	if err = p.parse(); err != nil {
		return err
	}
	w := bufio.NewWriter(cmd.OutOrStdout())
	defer w.Flush()
	_, err = printer.Printer{}.PrintLedger(w, p.builder.Build())
	return err
}

type parser struct {
	reader  *csv.Reader
	account *accounts.Account
	builder *ledger.Builder
}

func (p *parser) parse() error {
	p.reader.TrimLeadingSpace = true
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
	if ok, err := p.parseBooking(r); ok || err != nil {
		return err
	}
	return nil
}

var dateRegex = regexp.MustCompile(`\d\d.\d\d.\d\d\d\d`)

var replacer = strings.NewReplacer("CHF", "", "'", "")

func (p *parser) parseBooking(r []string) (bool, error) {
	if !dateRegex.MatchString(r[0]) || !dateRegex.MatchString(r[1]) {
		return false, nil
	}
	if len(r) != 11 {
		return false, fmt.Errorf("expected 11 items, got %v", r)
	}
	var words []string
	for _, i := range []int{2, 4, 5, 6, 7, 8} {
		s := strings.TrimSpace(r[i])
		if len(s) > 0 {
			words = append(words, s)
		}
	}
	var (
		err  error
		desc = strings.Join(words, " ")
		amt  decimal.Decimal
		d    time.Time
	)
	if d, err = time.Parse("02.01.2006", r[0]); err != nil {
		return false, err
	}
	if amt, err = decimal.NewFromString(replacer.Replace(r[3])); err != nil {
		return false, err
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
