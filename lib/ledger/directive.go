package ledger

import (
	"fmt"
	"time"

	"github.com/sboehler/knut/lib/date"
	"github.com/sboehler/knut/lib/scanner"
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
	Amount, Value              decimal.Decimal
	Credit, Debit              *Account
	Commodity, TargetCommodity *Commodity
	Lot                        *Lot
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
	Range
	Date        time.Time
	Description string
	Tags        []Tag
	Postings    []Posting
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
	Date time.Time
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
	Period      date.Period
	T0, T1      time.Time
	Account     *Account
	Transaction Transaction
}

// Expand expands an accrual transaction.
func (a Accrual) Expand() []Transaction {
	var (
		t                                                                = a.Transaction
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

		result []Transaction
	)
	if crAccountMulti != drAccountMulti {
		for i, date := range dates {
			var a = amount
			if i == 0 {
				a = a.Add(rem)
			}
			result = append(result, Transaction{
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
		result = append(result, Transaction{
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

func isAL(a *Account) bool {
	return a.Type() == ASSETS || a.Type() == LIABILITIES
}

func isIE(a *Account) bool {
	return a.Type() == INCOME || a.Type() == EXPENSES
}

// Currency declares that a commodity is a currency.
type Currency struct {
	Range
	*Commodity
}
