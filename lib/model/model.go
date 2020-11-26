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

package model

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/sboehler/knut/lib/amount"
	"github.com/sboehler/knut/lib/model/accounts"
	"github.com/sboehler/knut/lib/model/commodities"

	"github.com/shopspring/decimal"
)

// Tag represents a tag for a transaction or booking.
type Tag string

// String pretty-prints a tag.
func (t Tag) String() string {
	return string(t)
}

// Directive is a directive in a journal.
type Directive struct {
	Pos  Range
	date time.Time
}

// NewDirective returns a new directive.
func NewDirective(pos Range, date time.Time) Directive {
	return Directive{pos, date}
}

// Position returns the position.
func (d Directive) Position() Range {
	return d.Pos
}

// Date returns the date.
func (d Directive) Date() time.Time {
	return d.date
}

// Open represents an open command.
type Open struct {
	Directive
	Account *accounts.Account
}

// WriteTo pretty-prints an open directive.
func (o Open) WriteTo(b io.Writer) (int64, error) {
	n, err := fmt.Fprintf(b, "%s open %s", o.Date().Format("2006-01-02"), o.Account)
	return int64(n), err
}

// Close represents a close command.
type Close struct {
	Directive
	Account *accounts.Account
}

// WriteTo pretty-prints a close directive.
func (c Close) WriteTo(b io.Writer) (int64, error) {
	n, err := fmt.Fprintf(b, "%s close %s", c.Date().Format("2006-01-02"), c.Account)
	return int64(n), err
}

// Posting represents a posting.
type Posting struct {
	Amount        amount.Amount
	Credit, Debit *accounts.Account
	Commodity     *commodities.Commodity
	Lot           *Lot
	Tag           *Tag
}

func leftPad(n int, s string) string {
	if len(s) > n {
		return s
	}
	b := strings.Builder{}
	for i := 0; i < n-len(s); i++ {
		b.WriteRune(' ')
	}
	b.WriteString(s)
	return b.String()
}

// WriteTo pretty-prints a posting.
func (t Posting) WriteTo(b io.Writer) (int64, error) {
	var n int64
	c, err := fmt.Fprintf(b, "%s %s %s %s", t.Credit.RightPad(), t.Debit.RightPad(), leftPad(10, t.Amount.Amount().String()), t.Commodity)
	n += int64(c)
	if err != nil {
		return n, err
	}
	if t.Lot != nil {
		c, err = b.Write([]byte{' '})
		n += int64(c)
		if err != nil {
			return n, err
		}
		d, err := t.Lot.WriteTo(b)
		n += d
		if err != nil {
			return n, err
		}
	}
	if t.Tag != nil {
		c, err = fmt.Fprintf(b, " %s", t.Tag)
		n += int64(c)
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

// NewPosting creates a new posting from the given parameters. If amount is negative, it
// will be inverted and the accounts reversed.
func NewPosting(crAccount, drAccount *accounts.Account, commodity *commodities.Commodity, amt decimal.Decimal, tag *Tag) *Posting {
	if amt.IsNegative() {
		crAccount, drAccount = drAccount, crAccount
		amt = amt.Neg()
	}
	return &Posting{
		Credit:    crAccount,
		Debit:     drAccount,
		Amount:    amount.New(amt, nil),
		Commodity: commodity,
		Tag:       tag,
	}
}

// Lot represents a lot.
type Lot struct {
	Date      time.Time
	Label     string
	Price     float64
	Commodity *commodities.Commodity
}

// WriteTo pretty-prints a posting.
func (l Lot) WriteTo(b io.Writer) (int64, error) {
	var n int64
	c, err := fmt.Fprintf(b, "{ %g %s, %s ", l.Price, l.Commodity, l.Date.Format("2006-01-02"))
	n += int64(c)
	if err != nil {
		return int64(n), err
	}
	if len(l.Label) > 0 {
		c, err = fmt.Fprintf(b, "%s ", l.Label)
		n += int64(c)
		if err != nil {
			return n, err
		}
	}
	c, err = fmt.Fprint(b, "}")
	n += int64(c)
	if err != nil {
		return n, err
	}
	return n, nil
}

// Transaction represents a transaction.
type Transaction struct {
	Directive
	Description string
	Tags        []Tag
	Postings    []*Posting
}

// WriteTo pretty-prints a transaction.
func (t Transaction) WriteTo(b io.Writer) (int64, error) {
	var n int64
	c, err := fmt.Fprintf(b, `%s "%s"`, t.Date().Format("2006-01-02"), t.Description)
	n += int64(c)
	if err != nil {
		return n, err
	}
	for _, tag := range t.Tags {
		c, err := fmt.Fprintf(b, " %s", tag)
		n += int64(c)
		if err != nil {
			return n, err
		}
	}
	c, err = fmt.Fprint(b, "\n")
	n += int64(c)
	if err != nil {
		return n, err
	}
	for _, p := range t.Postings {
		d, err := p.WriteTo(b)
		n += int64(d)
		if err != nil {
			return n, err
		}
		c, err = fmt.Fprint(b, "\n")
		n += int64(c)
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

// Price represents a price command.
type Price struct {
	Directive
	Commodity *commodities.Commodity
	Target    *commodities.Commodity
	Price     float64
}

// WriteTo pretty-prints a Price directive.
func (p Price) WriteTo(w io.Writer) (int64, error) {
	n, err := fmt.Fprintf(w, "%s price %s %g %s", p.Date().Format("2006-01-02"), p.Commodity, p.Price, p.Target)
	return int64(n), err
}

// Include represents an include directive.
type Include struct {
	Directive
	Path string
}

// WriteTo pretty-prints an include directive
func (i Include) WriteTo(w io.Writer) (int64, error) {
	n, err := fmt.Fprintf(w, `include "%s"`, i.Path)
	return int64(n), err
}

// Assertion represents a balance assertion.
type Assertion struct {
	Directive
	Account   *accounts.Account
	Amount    decimal.Decimal
	Commodity *commodities.Commodity
}

// WriteTo pretty-prints an assertion directive.
func (a Assertion) WriteTo(w io.Writer) (int64, error) {
	n, err := fmt.Fprintf(w, "%s balance %s %s %s", a.Date().Format("2006-01-02"), a.Account, a.Amount, a.Commodity)
	return int64(n), err
}

func checkTrx(t *Transaction) error {
	if len(t.Postings)%2 == 1 {
		return fmt.Errorf("%v: Uneven number of postings", t.Position())
	}
	m := map[string]int{}
	for _, p := range t.Postings {
		m[p.Amount.Amount().Abs().String()]++
	}
	if len(m) != len(t.Postings)/2 {
		return fmt.Errorf("%v: Strange map %v", t.Position(), m)
	}
	for _, c := range m {
		if c != 2 {
			return fmt.Errorf("%v: Invalid count %v", t.Position(), m)
		}
	}
	return nil
}

// CommodityAccount represents a position.
type CommodityAccount struct {
	account   *accounts.Account
	commodity *commodities.Commodity
}

// NewCommodityAccount creates a new position
func NewCommodityAccount(a *accounts.Account, c *commodities.Commodity) CommodityAccount {
	return CommodityAccount{a, c}
}

// Account returns the account.
func (p CommodityAccount) Account() *accounts.Account {
	return p.account
}

// Commodity returns the commodity.
func (p CommodityAccount) Commodity() *commodities.Commodity {
	return p.commodity
}

// Less establishes a partial ordering of commodity accounts.
func (p CommodityAccount) Less(p1 CommodityAccount) bool {
	if p.account.Type() != p1.account.Type() {
		return p.account.Type() < p1.account.Type()
	}
	if p.account.String() != p1.account.String() {
		return p.account.String() < p1.account.String()
	}
	return p.commodity.String() < p1.commodity.String()
}

// Position is a position.
type Position struct {
	CommodityAccount
	Amounts amount.Vec
}

// Range describes a range in the source code.
type Range struct {
	Start int
	End   int
}
