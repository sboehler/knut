package model

import (
	"strings"
	"time"

	"github.com/sboehler/knut/lib/common/compare"
	"github.com/sboehler/knut/lib/syntax"
	"github.com/shopspring/decimal"
)

// Commodity represents a currency or security.
type Commodity struct {
	name       string
	IsCurrency bool
}

func (c Commodity) Name() string {
	return c.name
}

func (c Commodity) String() string {
	return c.name
}

// AccountType is the type of an account.
type AccountType int

const (
	// ASSETS represents an asset account.
	ASSETS AccountType = iota
	// LIABILITIES represents a liability account.
	LIABILITIES
	// EQUITY represents an equity account.
	EQUITY
	// INCOME represents an income account.
	INCOME
	// EXPENSES represents an expenses account.
	EXPENSES
)

func (t AccountType) String() string {
	switch t {
	case ASSETS:
		return "Assets"
	case LIABILITIES:
		return "Liabilities"
	case EQUITY:
		return "Equity"
	case INCOME:
		return "Income"
	case EXPENSES:
		return "Expenses"
	}
	return ""
}

// AccountTypes is an array with the ordered accont types.
var AccountTypes = []AccountType{ASSETS, LIABILITIES, EQUITY, INCOME, EXPENSES}

var accountTypes = map[string]AccountType{
	"Assets":      ASSETS,
	"Liabilities": LIABILITIES,
	"Equity":      EQUITY,
	"Expenses":    EXPENSES,
	"Income":      INCOME,
}

// Account represents an account which can be used in bookings.
type Account struct {
	accountType AccountType
	name        string
	segment     string
	level       int
}

// Split returns the account name split into segments.
func (a *Account) Split() []string {
	return strings.Split(a.name, ":")
}

// Name returns the name of this account.
func (a Account) Name() string {
	return a.name
}

// Segment returns the name of this account.
func (a Account) Segment() string {
	return a.segment
}

// Type returns the account type.
func (a Account) Type() AccountType {
	return a.accountType
}

// IsAL returns whether this account is an asset or liability account.
func (a Account) IsAL() bool {
	return a.accountType == ASSETS || a.accountType == LIABILITIES
}

// IsIE returns whether this account is an income or expense account.
func (a Account) IsIE() bool {
	return a.accountType == EXPENSES || a.accountType == INCOME
}

func (a Account) String() string {
	return a.name
}

// Directive is an element in a journal.
type Directive any

var (
	_ Directive = (*Assertion)(nil)
	_ Directive = (*Close)(nil)
	_ Directive = (*Open)(nil)
	_ Directive = (*Price)(nil)
	_ Directive = (*Transaction)(nil)
)

// Open represents an open command.
type Open struct {
	Src     *syntax.Open
	Date    time.Time
	Account *Account
}

// Close represents a close command.
type Close struct {
	Src     *syntax.Close
	Date    time.Time
	Account *Account
}

// Posting represents a posting.
type Posting struct {
	Src            *syntax.Booking
	Amount, Value  decimal.Decimal
	Account, Other *Account
	Commodity      *Commodity
}

// Less determines an order on postings.
func ComparePostings(p, p2 *Posting) compare.Order {
	if o := CompareAccounts(p.Account, p2.Account); o != compare.Equal {
		return o
	}
	if o := CompareAccounts(p.Other, p2.Other); o != compare.Equal {
		return o
	}
	if o := compare.Decimal(p.Amount, p2.Amount); o != compare.Equal {
		return o
	}
	if o := compare.Decimal(p.Value, p2.Value); o != compare.Equal {
		return o
	}
	return compare.Ordered(p.Commodity.Name(), p2.Commodity.Name())
}

// Transaction represents a transaction.
type Transaction struct {
	Src         *syntax.Transaction
	Date        time.Time
	Description string
	Postings    []*Posting
	Targets     []*Commodity
}

// Price represents a price command.
type Price struct {
	Src       *syntax.Price
	Date      time.Time
	Commodity *Commodity
	Target    *Commodity
	Price     decimal.Decimal
}

// Assertion represents a balance assertion.
type Assertion struct {
	Src       *syntax.Assertion
	Date      time.Time
	Account   *Account
	Amount    decimal.Decimal
	Commodity *Commodity
}
