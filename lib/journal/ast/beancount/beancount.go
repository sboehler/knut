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
	"sort"
	"strings"

	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
	"github.com/sboehler/knut/lib/journal/ast/printer"
	"github.com/shopspring/decimal"
)

// Transcode transcodes the given ledger to beancount.
func Transcode(w io.Writer, l []*ast.Day, c *journal.Commodity) error {
	if _, err := fmt.Fprintf(w, `option "operating_currency" "%s"`, c); err != nil {
		return err
	}
	if _, err := io.WriteString(w, "\n\n"); err != nil {
		return err
	}
	var p printer.Printer
	var openValAccounts = make(map[*journal.Account]bool)
	for _, day := range l {
		for _, open := range day.Openings {
			if _, err := p.PrintDirective(w, open); err != nil {
				return err
			}
			if _, err := io.WriteString(w, "\n\n"); err != nil {
				return err
			}
		}
		sort.Slice(day.Transactions, func(i, j int) bool {
			return day.Transactions[i].Less(day.Transactions[j])
		})

		for _, trx := range day.Transactions {
			for _, pst := range trx.Postings {
				if strings.HasPrefix(pst.Credit.Name(), "Equity:Valuation:") && !openValAccounts[pst.Credit] {
					openValAccounts[pst.Credit] = true
					if _, err := p.PrintDirective(w, &ast.Open{Date: trx.Date, Account: pst.Credit}); err != nil {
						return err
					}
					if _, err := io.WriteString(w, "\n\n"); err != nil {
						return err
					}
				}
				if strings.HasPrefix(pst.Debit.Name(), "Equity:Valuation:") && !openValAccounts[pst.Debit] {
					openValAccounts[pst.Debit] = true
					if _, err := p.PrintDirective(w, &ast.Open{Date: trx.Date, Account: pst.Debit}); err != nil {
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
			if _, err := p.PrintDirective(w, close); err != nil {
				return err
			}
			if _, err := io.WriteString(w, "\n\n"); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeTrx(w io.Writer, t *ast.Transaction, c *journal.Commodity) error {
	if _, err := fmt.Fprintf(w, `%s * "%s"`, t.Date.Format("2006-01-02"), t.Description); err != nil {
		return err
	}
	for _, tag := range t.Tags {
		if _, err := fmt.Fprintf(w, " %s", tag); err != nil {
			return err
		}
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
func writePosting(w io.Writer, p ast.Posting, c *journal.Commodity) error {
	var amt decimal.Decimal
	if c == nil {
		amt = p.Amount
	} else {
		amt = p.Value
	}
	if _, err := fmt.Fprintf(w, "  %s %s %s", p.Credit, amt.Neg(), stripNonAlphanum(c)); err != nil {
		return err
	}
	if _, err := io.WriteString(w, "\n"); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "  %s %s %s", p.Debit, amt, stripNonAlphanum(c)); err != nil {
		return err
	}
	if _, err := io.WriteString(w, "\n"); err != nil {
		return err
	}
	return nil
}

var regex = regexp.MustCompile("[^a-zA-Z]")

func stripNonAlphanum(c *journal.Commodity) string {
	return regex.ReplaceAllString(c.String(), "X")
}
