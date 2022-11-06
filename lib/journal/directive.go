package journal

import (
	"fmt"
	"time"

	"github.com/sboehler/knut/lib/common/compare"
	"github.com/sboehler/knut/lib/common/date"
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
	Credit, Debit *Account
	Commodity     *Commodity
	Targets       []*Commodity
	Lot           *Lot
}

// NewPosting creates a new posting from the given parameters. If amount is negative, it
// will be inverted and the accounts reversed.
func NewPosting(crAccount, drAccount *Account, commodity *Commodity, amt decimal.Decimal) Posting {
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
func PostingWithTargets(crAccount, drAccount *Account, commodity *Commodity, amt decimal.Decimal, targets []*Commodity) Posting {
	p := NewPosting(crAccount, drAccount, commodity, amt)
	p.Targets = targets
	return p
}

// NewValuePosting creates a value adjustment posting.
func NewValuePosting(crAccount, drAccount *Account, commodity *Commodity, val decimal.Decimal, targets []*Commodity) Posting {
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

func (pst *Posting) Accounts() []*Account {
	return []*Account{pst.Credit, pst.Debit}
}

// Less determines an order on postings.
func ComparePostings(p Posting, p2 Posting) compare.Order {
	if o := CompareAccounts(p.Credit, p2.Credit); o != compare.Equal {
		return o
	}
	if o := CompareAccounts(p.Debit, p2.Debit); o != compare.Equal {
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
	Postings    []Posting
	Accrual     *Accrual
}

// Position returns the source location.
func (t Transaction) Position() Range {
	return t.Range
}

// Commodities returns the commodities in this transaction.
func (t Transaction) Commodities() map[*Commodity]bool {
	var res = make(map[*Commodity]bool)
	for _, pst := range t.Postings {
		res[pst.Commodity] = true
	}
	return res
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
	Postings    []Posting
	Accrual     *Accrual
}

// Build builds a transactions.
func (tb TransactionBuilder) Build() *Transaction {
	compare.Sort(tb.Postings, ComparePostings)
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
	T0, T1   time.Time
	Account  *Account
}

// Expand expands an accrual transaction.
func (a Accrual) Expand(t *Transaction) []*Transaction {
	var (
		posting                                                          = t.Postings[0]
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
				Tags:        t.Tags,
				Description: fmt.Sprintf("%s (accrual %d/%d)", t.Description, i+1, len(periods)),
				Postings: []Posting{
					NewPosting(crAccountMulti, drAccountMulti, posting.Commodity, a),
				},
			}.Build())
		}
	}
	if crAccountSingle != drAccountSingle {
		result = append(result, TransactionBuilder{
			Range:       t.Position(),
			Date:        t.Date,
			Tags:        t.Tags,
			Description: t.Description,
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
	*Commodity
}
