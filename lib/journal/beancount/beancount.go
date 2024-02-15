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

package beancount

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/sboehler/knut/lib/common/compare"
	"github.com/sboehler/knut/lib/common/set"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/printer"
	"github.com/sboehler/knut/lib/model"
	"github.com/sboehler/knut/lib/model/transaction"
	"github.com/shopspring/decimal"
)

// Transcode transcodes the given journal to beancount.
func Transcode(w io.Writer, j *journal.Journal, c *model.Commodity) error {
	if _, err := fmt.Fprintf(w, `option "operating_currency" "%s"`, c.Name()); err != nil {
		return err
	}
	if _, err := io.WriteString(w, "\n\n"); err != nil {
		return err
	}
	p := printer.New(w)
	openValAccounts := set.New[*model.Account]()
	for _, day := range j.Days {
		for _, open := range day.Openings {
			if _, err := p.PrintDirective(open); err != nil {
				return err
			}
			if _, err := io.WriteString(w, "\n\n"); err != nil {
				return err
			}
		}
		compare.Sort(day.Transactions, transaction.Compare)

		for _, trx := range day.Transactions {
			for _, pst := range trx.Postings {
				if strings.HasPrefix(pst.Account.Name(), "Equity:Valuation:") && !openValAccounts.Has(pst.Account) {
					openValAccounts.Add(pst.Account)
					if _, err := p.PrintDirective(&model.Open{Date: trx.Date, Account: pst.Account}); err != nil {
						return err
					}
					if _, err := io.WriteString(w, "\n\n"); err != nil {
						return err
					}
				}
			}
		}
		for _, trx := range day.Transactions {
			if err := writeTrx(w, trx, c); err != nil {
				return err
			}
		}
		for _, close := range day.Closings {
			if _, err := p.PrintDirective(close); err != nil {
				return err
			}
			if _, err := io.WriteString(w, "\n\n"); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeTrx(w io.Writer, t *model.Transaction, c *model.Commodity) error {
	if _, err := fmt.Fprintf(w, `%s * "%s"`, t.Date.Format("2006-01-02"), t.Description); err != nil {
		return err
	}
	if _, err := io.WriteString(w, "\n"); err != nil {
		return err
	}
	for _, p := range t.Postings {
		if err := writePosting(w, p, c); err != nil {
			return err
		}
	}
	_, err := io.WriteString(w, "\n")
	return err
}

// WriteTo pretty-prints a posting.
func writePosting(w io.Writer, p *model.Posting, c *model.Commodity) error {
	var quantity decimal.Decimal
	if c == nil {
		quantity = p.Quantity
	} else {
		quantity = p.Value
	}
	if _, err := fmt.Fprintf(w, "  %s %s %s", p.Account.Name(), quantity, stripNonAlphanum(c)); err != nil {
		return err
	}
	if _, err := io.WriteString(w, "\n"); err != nil {
		return err
	}
	return nil
}

var regex = regexp.MustCompile("[^a-zA-Z]")

func stripNonAlphanum(c *model.Commodity) string {
	return regex.ReplaceAllString(c.Name(), "X")
}
