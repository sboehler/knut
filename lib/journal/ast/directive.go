package ast

import (
	"fmt"
	"time"

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
	_ Directive = (*Accrual)(nil)
	_ Directive = (*Assertion)(nil)
	_ Directive = (*Close)(nil)
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
	Amount, Value              decimal.Decimal
	Credit, Debit              *journal.Account
	Commodity, TargetCommodity *journal.Commodity
	Lot                        *Lot
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

// Less determines an order on postings.
func (p Posting) Less(p2 Posting) bool {
	if p.Credit.Name() != p2.Credit.Name() {
		return p.Credit.Name() < p2.Credit.Name()
	}
	if p.Debit.Name() != p2.Debit.Name() {
		return p.Debit.Name() < p2.Debit.Name()
	}
	if !p.Amount.Equal(p2.Amount) {
		return p.Amount.LessThan(p2.Amount)
	}
	return p.Commodity.String() < p2.Commodity.String()
}

// Equal determines a measure of equality.
func (p Posting) Equal(p2 Posting) bool {
	return p.Credit == p2.Credit &&
		p.Debit == p2.Debit &&
		p.Amount.Equals(p2.Amount) &&
		p.Commodity == p2.Commodity
}

// Matches returns whether this filter matches the given Posting.
func (p Posting) Matches(b journal.Filter) bool {
	return (b.MatchAccount(p.Credit) || b.MatchAccount(p.Debit)) && b.MatchCommodity(p.Commodity)
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
	Range
	Date        time.Time
	Description string
	Tags        []Tag
	Postings    []Posting
	AddOns      []interface{}
}

// Clone clones a transaction.
func (t Transaction) Clone() *Transaction {
	var (
		tags     = make([]Tag, len(t.Tags))
		postings = make([]Posting, len(t.Postings))
		addOns   = make([]interface{}, len(t.AddOns))
	)
	copy(tags, t.Tags)
	copy(postings, t.Postings)
	copy(addOns, t.AddOns)
	return &Transaction{
		Range:       t.Range,
		Date:        t.Date,
		Description: t.Description,
		Tags:        tags,
		Postings:    postings,
		AddOns:      addOns,
	}
}

// Commodities returns the commodities in this transaction.
func (t Transaction) Commodities() map[*journal.Commodity]bool {
	var res = make(map[*journal.Commodity]bool)
	for _, pst := range t.Postings {
		res[pst.Commodity] = true
	}
	return res
}

// Less defines an order on transactions.
func (t *Transaction) Less(t2 *Transaction) bool {
	if !t.Date.Equal(t2.Date) {
		return t.Date.Before(t2.Date)
	}
	if t.Description != t2.Description {
		return t.Description < t2.Description
	}
	var i int
	for i < len(t.Postings) && i < len(t2.Postings) {
		if !t.Postings[i].Equal(t2.Postings[i]) {
			return t.Postings[i].Less(t2.Postings[i])
		}
	}
	return len(t.Postings) < len(t2.Postings)
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
	Date time.Time
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
	Period  date.Period
	T0, T1  time.Time
	Account *journal.Account
}

// Expand expands an accrual transaction.
func (a Accrual) Expand(t *Transaction) []*Transaction {
	var (
		posting                                                          = t.Postings[0]
		crAccountSingle, drAccountSingle, crAccountMulti, drAccountMulti = a.Account, a.Account, a.Account, a.Account
	)
	switch {
	case isAL(posting.Credit) && isIE(posting.Debit):
		crAccountSingle = posting.Credit
		drAccountMulti = posting.Debit
	case isIE(posting.Credit) && isAL(posting.Debit):
		crAccountMulti = posting.Credit
		drAccountSingle = posting.Debit
	case isIE(posting.Credit) && isIE(posting.Debit):
		crAccountMulti = posting.Credit
		drAccountMulti = posting.Debit
	default:
		crAccountSingle = posting.Credit
		drAccountSingle = posting.Debit
	}
	var (
		dates       = date.Series(a.T0, a.T1, a.Period)[1:]
		amount, rem = posting.Amount.QuoRem(decimal.NewFromInt(int64(len(dates))), 1)

		result []*Transaction
	)
	if crAccountMulti != drAccountMulti {
		for i, date := range dates {
			var a = amount
			if i == 0 {
				a = a.Add(rem)
			}
			result = append(result, &Transaction{
				Range:       t.Range,
				Date:        date,
				Tags:        t.Tags,
				Description: fmt.Sprintf("%s (accrual %d/%d)", t.Description, i+1, len(dates)),
				Postings: []Posting{
					NewPosting(crAccountMulti, drAccountMulti, posting.Commodity, a),
				},
			})
		}
	}
	if crAccountSingle != drAccountSingle {
		result = append(result, &Transaction{
			Range:       t.Range,
			Date:        t.Date,
			Tags:        t.Tags,
			Description: t.Description,
			Postings: []Posting{
				NewPosting(crAccountSingle, drAccountSingle, posting.Commodity, posting.Amount),
			},
		})

	}
	return result
}

func isAL(a *journal.Account) bool {
	return a.Type() == journal.ASSETS || a.Type() == journal.LIABILITIES
}

func isIE(a *journal.Account) bool {
	return a.Type() == journal.INCOME || a.Type() == journal.EXPENSES
}

// Currency declares that a commodity is a currency.
type Currency struct {
	Range
	*journal.Commodity
}
