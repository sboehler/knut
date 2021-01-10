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

// Open represents an open command.
type Open struct {
	Pos     Range
	Date    time.Time
	Account *accounts.Account
}

// Position returns the position.
func (o Open) Position() Range {
	return o.Pos
}

// Close represents a close command.
type Close struct {
	Pos     Range
	Date    time.Time
	Account *accounts.Account
}

// Position returns the position.
func (c Close) Position() Range {
	return c.Pos
}

// Posting represents a posting.
type Posting struct {
	Amount        amount.Amount
	Credit, Debit *accounts.Account
	Commodity     *commodities.Commodity
	Lot           *Lot
	Tag           *Tag
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

// Transaction represents a transaction.
type Transaction struct {
	Pos         Range
	Date        time.Time
	Description string
	Tags        []Tag
	Postings    []*Posting
}

// Position returns the Position.
func (t Transaction) Position() Range {
	return t.Pos
}

// Price represents a price command.
type Price struct {
	Pos       Range
	Date      time.Time
	Commodity *commodities.Commodity
	Target    *commodities.Commodity
	Price     float64
}

// Position returns the Range.
func (p Price) Position() Range {
	return p.Pos
}

// Include represents an include directive.
type Include struct {
	Pos  Range
	Date time.Time
	Path string
}

// Position returns the Range.
func (i Include) Position() Range {
	return i.Pos
}

// Assertion represents a balance assertion.
type Assertion struct {
	Pos       Range
	Date      time.Time
	Account   *accounts.Account
	Amount    decimal.Decimal
	Commodity *commodities.Commodity
}

// Position returns the Range.
func (a Assertion) Position() Range {
	return a.Pos
}

// Value represents a value directive.
type Value struct {
	Pos       Range
	Date      time.Time
	Account   *accounts.Account
	Amount    decimal.Decimal
	Commodity *commodities.Commodity
}

// Position returns the Range.
func (v Value) Position() Range {
	return v.Pos
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
	Start, End FilePosition
}

// FilePosition is a position of a character in a text file.
type FilePosition struct {
	Path                           string
	BytePos, RunePos, Line, Column int
}

func (p FilePosition) String() string {
	return fmt.Sprintf("%s:%d:%d", p.Path, p.Line, p.Column)
}
