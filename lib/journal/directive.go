package journal

import (
	"fmt"
	"time"

	"github.com/sboehler/knut/lib/common/compare"
	"github.com/sboehler/knut/lib/common/date"
	"github.com/sboehler/knut/lib/common/slice"
	"github.com/sboehler/knut/lib/journal/scanner"
	"github.com/shopspring/decimal"
)

// Range describes a range of locations in a file.
type Range struct {
	Path       string
	Start, End scanner.Location
}

// Position returns the Range itself.
func (r Range) Position() Range {
	return r
}

// Directive is an element in a journal with a position.
type Directive interface {
	Position() Range
}

var (
	_ Directive = (*Assertion)(nil)
	_ Directive = (*Close)(nil)
	_ Directive = (*Currency)(nil)
	_ Directive = (*Include)(nil)
	_ Directive = (*Open)(nil)
	_ Directive = (*Price)(nil)
	_ Directive = (*Transaction)(nil)
	_ Directive = (*Value)(nil)
)

// Open represents an open command.
type Open struct {
	Range
	Date    time.Time
	Account *Account
}

// Close represents a close command.
type Close struct {
	Range
	Date    time.Time
	Account *Account
}

// Posting represents a posting.
type Posting struct {
	Amount, Value decimal.Decimal
	//Credit, Debit *Account
	Account, Other *Account
	Commodity      *Commodity
	Targets        []*Commodity
	Lot            *Lot
}

type PostingBuilder struct {
	Amount, Value decimal.Decimal
	Credit, Debit *Account
	Commodity     *Commodity
	Targets       []*Commodity
	Lot           *Lot
}

func (pb PostingBuilder) Build() [2]*Posting {
	if pb.Amount.IsNegative() || pb.Amount.IsZero() && pb.Value.IsNegative() {
		pb.Credit, pb.Debit, pb.Amount, pb.Value = pb.Debit, pb.Credit, pb.Amount.Neg(), pb.Value.Neg()
	}
	return [2]*Posting{
		{
			Account:   pb.Credit,
			Other:     pb.Debit,
			Commodity: pb.Commodity,
			Amount:    pb.Amount.Neg(),
			Value:     pb.Value.Neg(),
			Targets:   pb.Targets,
			Lot:       pb.Lot,
		},
		{
			Account:   pb.Debit,
			Other:     pb.Credit,
			Commodity: pb.Commodity,
			Amount:    pb.Amount,
			Value:     pb.Value,
			Targets:   pb.Targets,
			Lot:       pb.Lot,
		},
	}
}

func (pb PostingBuilder) Singleton() []*Posting {
	return slice.Concat(pb.Build())
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
	if o := compare.Ordered(p.Commodity.Name(), p2.Commodity.Name()); o != compare.Equal {
		return o
	}
	return compare.Ordered(len(p.Targets), len(p2.Targets))
}

// Lot represents a lot.
type Lot struct {
	Date      time.Time
	Label     string
	Price     float64
	Commodity *Commodity
}

// Tag represents a tag for a transaction or booking.
type Tag string

// Transaction represents a transaction.
type Transaction struct {
	Range       Range
	Date        time.Time
	Description string
	Tags        []Tag
	Postings    []*Posting
	Accrual     *Accrual
}

// Position returns the source location.
func (t Transaction) Position() Range {
	return t.Range
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
	Range       Range
	Date        time.Time
	Description string
	Tags        []Tag
	Postings    []*Posting
	Accrual     *Accrual
}

// Build builds a transactions.
func (tb TransactionBuilder) Build() *Transaction {
	// compare.Sort(tb.Postings, ComparePostings)
	return &Transaction{
		Range:       tb.Range,
		Date:        tb.Date,
		Description: tb.Description,
		Tags:        tb.Tags,
		Postings:    tb.Postings,
		Accrual:     tb.Accrual,
	}
}

// Price represents a price command.
type Price struct {
	Range
	Date      time.Time
	Commodity *Commodity
	Target    *Commodity
	Price     decimal.Decimal
}

// Include represents an include directive.
type Include struct {
	Range
	Path string
}

// Assertion represents a balance assertion.
type Assertion struct {
	Range
	Date      time.Time
	Account   *Account
	Amount    decimal.Decimal
	Commodity *Commodity
}

// Value represents a value directive.
type Value struct {
	Range
	Date      time.Time
	Account   *Account
	Amount    decimal.Decimal
	Commodity *Commodity
}

// Accrual represents an accrual.
type Accrual struct {
	Range
	Interval date.Interval
	Period   date.Period
	Account  *Account
}

// Expand expands an accrual transaction.
func (a Accrual) Expand(t *Transaction) []*Transaction {
	var (
		result []*Transaction
	)
	for _, p := range t.Postings {
		if p.Account.IsAL() {
			result = append(result, TransactionBuilder{
				Range:       t.Position(),
				Date:        t.Date,
				Tags:        t.Tags,
				Description: t.Description,
				Postings: PostingBuilder{
					Credit:    t.Accrual.Account,
					Debit:     p.Account,
					Commodity: p.Commodity,
					Amount:    p.Amount,
				}.Singleton(),
			}.Build())
		}
		if p.Account.IsIE() {
			dates := a.Period.Dates(a.Interval, 0)
			amount, rem := p.Amount.QuoRem(decimal.NewFromInt(int64(len(dates))), 1)
			for i, dt := range dates {
				a := amount
				if i == 0 {
					a = a.Add(rem)
				}
				result = append(result, TransactionBuilder{
					Range:       t.Position(),
					Date:        dt,
					Tags:        t.Tags,
					Description: fmt.Sprintf("%s (accrual %d/%d)", t.Description, i+1, len(dates)),
					Postings: PostingBuilder{
						Credit:    t.Accrual.Account,
						Debit:     p.Account,
						Commodity: p.Commodity,
						Amount:    a,
					}.Singleton(),
				}.Build())
			}
		}
	}
	return result
}

// Currency declares that a commodity is a currency.
type Currency struct {
	Range
	Date time.Time
	*Commodity
}
