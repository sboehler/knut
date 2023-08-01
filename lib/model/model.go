package model

import (
	"fmt"
	"strings"
	"time"

	"github.com/sboehler/knut/lib/common/compare"
	"github.com/sboehler/knut/lib/common/date"
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

func CompareAccountTypes(t1, t2 AccountType) compare.Order {
	if t1 == t2 {
		return compare.Equal
	}
	if t1 < t2 {
		return compare.Smaller
	}
	return compare.Greater
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

func CompareAccounts(a1, a2 *Account) compare.Order {
	o := CompareAccountTypes(a1.accountType, a2.accountType)
	if o != compare.Equal {
		return o
	}
	return compare.Ordered(a1.name, a2.name)
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

type PostingBuilder struct {
	Src           *syntax.Booking
	Amount, Value decimal.Decimal
	Credit, Debit *Account
	Commodity     *Commodity
}

func (pb PostingBuilder) Build() []*Posting {
	if pb.Amount.IsNegative() || pb.Amount.IsZero() && pb.Value.IsNegative() {
		pb.Credit, pb.Debit, pb.Amount, pb.Value = pb.Debit, pb.Credit, pb.Amount.Neg(), pb.Value.Neg()
	}
	return []*Posting{
		{
			Src:       pb.Src,
			Account:   pb.Credit,
			Other:     pb.Debit,
			Commodity: pb.Commodity,
			Amount:    pb.Amount.Neg(),
			Value:     pb.Value.Neg(),
		},
		{
			Src:       pb.Src,
			Account:   pb.Debit,
			Other:     pb.Credit,
			Commodity: pb.Commodity,
			Amount:    pb.Amount,
			Value:     pb.Value,
		},
	}
}

type PostingBuilders []PostingBuilder

func (pbs PostingBuilders) Build() []*Posting {
	res := make([]*Posting, 0, 2*len(pbs))
	for _, pb := range pbs {
		res = append(res, pb.Build()...)
	}
	return res
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
	Accrual     *Accrual
	Targets     []*Commodity
}

// Less defines an order on transactions.
func CompareTransactions(t *Transaction, t2 *Transaction) compare.Order {
	if o := compare.Time(t.Date, t2.Date); o != compare.Equal {
		return o
	}
	if o := compare.Ordered(t.Description, t2.Description); o != compare.Equal {
		return o
	}
	for i := 0; i < len(t.Postings) && i < len(t2.Postings); i++ {
		if o := ComparePostings(t.Postings[i], t2.Postings[i]); o != compare.Equal {
			return o
		}
	}
	return compare.Ordered(len(t.Postings), len(t2.Postings))
}

// TransactionBuilder builds transactions.
type TransactionBuilder struct {
	Src         *syntax.Transaction
	Date        time.Time
	Description string
	Postings    []*Posting
	Targets     []*Commodity
	Accrual     *Accrual
}

// Build builds a transactions.
func (tb TransactionBuilder) Build() *Transaction {
	return &Transaction{
		Src:         tb.Src,
		Date:        tb.Date,
		Description: tb.Description,
		Postings:    tb.Postings,
		Accrual:     tb.Accrual,
		Targets:     tb.Targets,
	}
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

// Accrual represents an accrual.
type Accrual struct {
	Src      *syntax.Accrual
	Interval date.Interval
	Period   date.Period
	Account  *Account
}

// Expand expands an accrual transaction.
func (a Accrual) Expand(t *Transaction) []*Transaction {
	var result []*Transaction
	for _, p := range t.Postings {
		if p.Account.IsAL() {
			result = append(result, TransactionBuilder{
				Src:         t.Src,
				Date:        t.Date,
				Description: t.Description,
				Postings: PostingBuilder{
					Credit:    t.Accrual.Account,
					Debit:     p.Account,
					Commodity: p.Commodity,
					Amount:    p.Amount,
				}.Build(),
			}.Build())
		}
		if p.Account.IsIE() {
			partition := date.NewPartition(a.Period, a.Interval, 0)
			amount, rem := p.Amount.QuoRem(decimal.NewFromInt(int64(partition.Size())), 1)
			for i, dt := range partition.EndDates() {
				a := amount
				if i == 0 {
					a = a.Add(rem)
				}
				result = append(result, TransactionBuilder{
					Src:         t.Src,
					Date:        dt,
					Description: fmt.Sprintf("%s (accrual %d/%d)", t.Description, i+1, partition.Size()),
					Postings: PostingBuilder{
						Credit:    t.Accrual.Account,
						Debit:     p.Account,
						Commodity: p.Commodity,
						Amount:    a,
					}.Build(),
				}.Build())
			}
		}
	}
	return result
}
