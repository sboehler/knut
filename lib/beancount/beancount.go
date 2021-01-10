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

package beancount

import (
	"fmt"
	"io"
	"regexp"

	"github.com/sboehler/knut/lib/ledger"
	"github.com/sboehler/knut/lib/model/accounts"
	"github.com/sboehler/knut/lib/model/commodities"
	"github.com/sboehler/knut/lib/printer"
)

// Transcode transcodes the given ledger to beancount.
func Transcode(w io.Writer, l ledger.Ledger, c *commodities.Commodity) error {
	if _, err := fmt.Fprintf(w, `option "operating_currency" "%s"`, c); err != nil {
		return err
	}
	if _, err := io.WriteString(w, "\n\n"); err != nil {
		return err
	}
	l[0].Openings = append(l[0].Openings,
		&ledger.Open{
			Date:    l[0].Date,
			Account: accounts.ValuationAccount(),
		},
		&ledger.Open{
			Date:    l[0].Date,
			Account: accounts.RetainedEarningsAccount(),
		},
	)
	p := printer.Printer{}
	for _, day := range l {
		for _, open := range day.Openings {
			if _, err := p.PrintDirective(w, open); err != nil {
				return err
			}
			if _, err := io.WriteString(w, "\n\n"); err != nil {
				return err
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

func writeTrx(w io.Writer, t *ledger.Transaction, c *commodities.Commodity) error {
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
func writePosting(w io.Writer, p *ledger.Posting, c *commodities.Commodity) error {
	if _, err := fmt.Fprintf(w, "  %s %s %s", p.Credit, p.Amount.Valuation(0).Neg(), stripNonAlphanum(c)); err != nil {
		return err
	}
	if _, err := io.WriteString(w, "\n"); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "  %s %s %s", p.Debit, p.Amount.Valuation(0), stripNonAlphanum(c)); err != nil {
		return err
	}
	if _, err := io.WriteString(w, "\n"); err != nil {
		return err
	}
	return nil
}

var regex = regexp.MustCompile("[^a-zA-Z]")

func stripNonAlphanum(c *commodities.Commodity) string {
	s := c.String()
	return regex.ReplaceAllString(s, "X")
}
