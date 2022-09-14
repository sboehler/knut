package ast

import (
	"fmt"
	"time"

	"github.com/sboehler/knut/lib/common/compare"
	"github.com/sboehler/knut/lib/common/date"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast/scanner"
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
	Account *journal.Account
}

// Close represents a close command.
type Close struct {
	Range
	Date    time.Time
	Account *journal.Account
}

// Posting represents a posting.
type Posting struct {
	Amount, Value decimal.Decimal
	Credit, Debit *journal.Account
	Commodity     *journal.Commodity
	Targets       []*journal.Commodity
	Lot           *Lot
}

// NewPosting creates a new posting from the given parameters. If amount is negative, it
// will be inverted and the accounts reversed.
func NewPosting(crAccount, drAccount *journal.Account, commodity *journal.Commodity, amt decimal.Decimal) Posting {
	if amt.IsNegative() {
		crAccount, drAccount = drAccount, crAccount
		amt = amt.Neg()
	}
	return Posting{
		Credit:    crAccount,
		Debit:     drAccount,
		Amount:    amt,
		Commodity: commodity,
	}
}

// PostingWithTargets creates a new posting from the given parameters. If amount is negative, it
// will be inverted and the accounts reversed.
func PostingWithTargets(crAccount, drAccount *journal.Account, commodity *journal.Commodity, amt decimal.Decimal, targets []*journal.Commodity) Posting {
	p := NewPosting(crAccount, drAccount, commodity, amt)
	p.Targets = targets
	return p
}

// NewValuePosting creates a value adjustment posting.
func NewValuePosting(crAccount, drAccount *journal.Account, commodity *journal.Commodity, val decimal.Decimal, targets []*journal.Commodity) Posting {
	if val.IsNegative() {
		crAccount, drAccount = drAccount, crAccount
		val = val.Neg()
	}
	return Posting{
		Credit:    crAccount,
		Debit:     drAccount,
		Value:     val,
		Commodity: commodity,
		Targets:   targets,
	}
}

// Less determines an order on postings.
func ComparePostings(p Posting, p2 Posting) compare.Order {
	if o := journal.CompareAccounts(p.Credit, p2.Credit); o != compare.Equal {
		return o
	}
	if o := journal.CompareAccounts(p.Debit, p2.Debit); o != compare.Equal {
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
	Commodity *journal.Commodity
}

// Tag represents a tag for a transaction or booking.
type Tag string

// Transaction represents a transaction.
type Transaction struct {
	rng         Range
	date        time.Time
	description string
	tags        []Tag
	postings    []Posting
	accrual     *Accrual
}

// Description returns the description.
func (t Transaction) Description() string {
	return t.description
}

// Date returns the transaction date.
func (t Transaction) Date() time.Time {
	return t.date
}

// Position returns the source location.
func (t Transaction) Position() Range {
	return t.rng
}

// Tags returns the tags.
func (t Transaction) Tags() []Tag {
	return t.tags
}

// Postings returns the postings.
func (t Transaction) Postings() []Posting {
	return t.postings
}

// Accrual returns the accrual.
func (t Transaction) Accrual() *Accrual {
	return t.accrual
}

// ToBuilder creates a new builder based on this transaction.
func (t Transaction) ToBuilder() TransactionBuilder {
	var (
		tags     = make([]Tag, len(t.tags))
		postings = make([]Posting, len(t.postings))
	)
	copy(tags, t.tags)
	copy(postings, t.postings)
	return TransactionBuilder{
		Range:       t.rng,
		Date:        t.date,
		Description: t.description,
		Tags:        tags,
		Postings:    postings,
		Accrual:     t.accrual,
	}
}

// Commodities returns the commodities in this transaction.
func (t Transaction) Commodities() map[*journal.Commodity]bool {
	var res = make(map[*journal.Commodity]bool)
	for _, pst := range t.postings {
		res[pst.Commodity] = true
	}
	return res
}

// Less defines an order on transactions.
func CompareTransactions(t *Transaction, t2 *Transaction) compare.Order {
	if o := compare.Time(t.date, t2.date); o != compare.Equal {
		return o
	}
	if o := compare.Ordered(t.description, t2.description); o != compare.Equal {
		return o
	}
	for i := 0; i < len(t.postings) && i < len(t2.postings); i++ {
		if o := ComparePostings(t.postings[i], t2.postings[i]); o != compare.Equal {
			return o
		}
	}
	return compare.Ordered(len(t.postings), len(t2.postings))
}

// TransactionBuilder builds transactions.
type TransactionBuilder struct {
	Range       Range
	Date        time.Time
	Description string
	Tags        []Tag
	Postings    []Posting
	Accrual     *Accrual
}

// Build builds a transactions.
func (tb TransactionBuilder) Build() *Transaction {
	compare.Sort(tb.Postings, ComparePostings)
	return &Transaction{
		rng:         tb.Range,
		date:        tb.Date,
		description: tb.Description,
		tags:        tb.Tags,
		postings:    tb.Postings,
		accrual:     tb.Accrual,
	}
}

// Price represents a price command.
type Price struct {
	Range
	Date      time.Time
	Commodity *journal.Commodity
	Target    *journal.Commodity
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
	Account   *journal.Account
	Amount    decimal.Decimal
	Commodity *journal.Commodity
}

// Value represents a value directive.
type Value struct {
	Range
	Date      time.Time
	Account   *journal.Account
	Amount    decimal.Decimal
	Commodity *journal.Commodity
}

// Accrual represents an accrual.
type Accrual struct {
	Range
	Interval date.Interval
	T0, T1   time.Time
	Account  *journal.Account
}

// Expand expands an accrual transaction.
func (a Accrual) Expand(t *Transaction) []*Transaction {
	var (
		posting                                                          = t.postings[0]
		crAccountSingle, drAccountSingle, crAccountMulti, drAccountMulti = a.Account, a.Account, a.Account, a.Account
	)
	switch {
	case posting.Credit.IsAL() && posting.Debit.IsIE():
		crAccountSingle = posting.Credit
		drAccountMulti = posting.Debit
	case posting.Credit.IsIE() && posting.Debit.IsAL():
		crAccountMulti = posting.Credit
		drAccountSingle = posting.Debit
	case posting.Credit.IsIE() && posting.Debit.IsIE():
		crAccountMulti = posting.Credit
		drAccountMulti = posting.Debit
	default:
		crAccountSingle = posting.Credit
		drAccountSingle = posting.Debit
	}
	var (
		periods     = date.Periods(a.T0, a.T1, a.Interval)
		amount, rem = posting.Amount.QuoRem(decimal.NewFromInt(int64(len(periods))), 1)

		result []*Transaction
	)
	if crAccountMulti != drAccountMulti {
		for i, period := range periods {
			var a = amount
			if i == 0 {
				a = a.Add(rem)
			}
			result = append(result, TransactionBuilder{
				Range:       t.Position(),
				Date:        period.End,
				Tags:        t.Tags(),
				Description: fmt.Sprintf("%s (accrual %d/%d)", t.Description(), i+1, len(periods)),
				Postings: []Posting{
					NewPosting(crAccountMulti, drAccountMulti, posting.Commodity, a),
				},
			}.Build())
		}
	}
	if crAccountSingle != drAccountSingle {
		result = append(result, TransactionBuilder{
			Range:       t.Position(),
			Date:        t.Date(),
			Tags:        t.Tags(),
			Description: t.description,
			Postings: []Posting{
				NewPosting(crAccountSingle, drAccountSingle, posting.Commodity, posting.Amount),
			},
		}.Build())

	}
	return result
}

// Currency declares that a commodity is a currency.
type Currency struct {
	Range
	Date time.Time
	*journal.Commodity
}
