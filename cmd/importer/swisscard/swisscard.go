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
	var cmd = &cobra.Command{
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
	var p = parser{
		reader:  csv.NewReader(f),
		builder: journal.NewBuilder(ctx),
	}
	if p.account, err = r.account.Value(ctx); err != nil {
		return err
	}
	if err = p.parse(); err != nil {
		return err
	}
	w := bufio.NewWriter(cmd.OutOrStdout())
	defer w.Flush()
	_, err = journal.NewPrinter().PrintLedger(w, p.builder.SortedDays())
	return err
}

type parser struct {
	reader  *csv.Reader
	account *journal.Account
	builder *journal.JournalBuilder
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
		chf  *journal.Commodity
		amt  decimal.Decimal
		d    time.Time
	)
	if d, err = time.Parse("02.01.2006", r[0]); err != nil {
		return false, err
	}
	if amt, err = decimal.NewFromString(replacer.Replace(r[3])); err != nil {
		return false, err
	}
	if chf, err = p.builder.Context.GetCommodity("CHF"); err != nil {
		return false, err
	}
	p.builder.AddTransaction(journal.TransactionBuilder{
		Date:        d,
		Description: desc,
		Postings: []*journal.Posting{
			journal.NewPosting(p.account, p.builder.Context.TBDAccount(), chf, amt),
		},
	}.Build())
	return true, nil
}
