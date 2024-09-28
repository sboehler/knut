package directives

import (
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

type Commodity struct{ Range }

type Account struct {
	Range
	Macro bool
}

type Date struct{ Range }

func (d Date) Parse() (time.Time, error) {
	date, err := time.Parse("2006-01-02", d.Extract())
	if err != nil {
		return date, Error{
			Message: "parsing date",
			Range:   d.Range,
			Wrapped: err,
		}
	}
	return date, nil
}

type Decimal struct{ Range }

func (d Decimal) Parse() (decimal.Decimal, error) {
	dec, err := decimal.NewFromString(d.Extract())
	if err != nil {
		return dec, Error{
			Message: "parsing date",
			Range:   d.Range,
			Wrapped: err,
		}
	}
	return dec, nil
}

type QuotedString struct {
	Range
	Content Range
}

type Booking struct {
	Range
	Credit, Debit Account
	Quantity      Decimal
	Commodity     Commodity
}

type Performance struct {
	Range
	Targets []Commodity
}

type Interval struct{ Range }

type Directive struct {
	Range
	Directive any
}

type File struct {
	Range
	Directives []Directive
}

type Accrual struct {
	Range
	Interval   Interval
	Start, End Date
	Account    Account
}

type Addons struct {
	Range
	Performance Performance
	Accrual     Accrual
}

type Transaction struct {
	Range
	Date        Date
	Description []QuotedString
	Bookings    []Booking
	Addons      Addons
}

type Open struct {
	Range
	Date    Date
	Account Account
}

type Close struct {
	Range
	Date    Date
	Account Account
}

type Assertion struct {
	Range
	Date     Date
	Balances []Balance
}

type Balance struct {
	Range
	Account   Account
	Quantity  Decimal
	Commodity Commodity
}

type Price struct {
	Range
	Date              Date
	Commodity, Target Commodity
	Price             Decimal
}

type Include struct {
	Range
	IncludePath QuotedString
}

type Range struct {
	Start, End int
	Path, Text string
}

func (r Range) Extract() string {
	return r.Text[r.Start:r.End]
}

func (r *Range) SetRange(r2 Range) {
	*r = r2
}

func (r Range) Length() int {
	return r.End - r.Start
}

func (r *Range) Extend(r2 Range) {
	if r.Start > r2.Start {
		r.Start = r2.Start
	}
	if r.End < r2.End {
		r.End = r2.End
	}
}

func SetRange[T any, P interface {
	*T
	SetRange(Range)
}](t P, r Range) T {
	t.SetRange(r)
	return *t
}

func (r Range) Empty() bool {
	return r.Start == r.End
}

func (r Range) Location() Location {
	loc := Location{Line: 1, Col: 1}
	for pos, ch := range r.Text {
		if pos == r.End {
			return loc
		}
		if ch == '\n' {
			loc.Line++
			loc.Col = 1
		} else {
			loc.Col++
		}
	}
	return loc
}

func (r Range) Context(previous int) []string {
	start := r.Start
	end := r.End
	for i := 0; i <= previous; i++ {
		start = r.firstOfLine(start)
	}
	end = r.lastOfLine(end)
	return strings.Split(r.Text[start:end], "\n")
}

func (r Range) firstOfLine(pos int) int {
	for pos > 0 && r.Text[pos-1] != '\n' {
		pos--
	}
	return pos
}

func (r Range) lastOfLine(pos int) int {
	for pos < len(r.Text) && r.Text[pos] != '\n' {
		pos++
	}
	return pos
}

type Location struct {
	Line, Col int
}

func (l Location) String() string {
	return fmt.Sprintf("%d:%d", l.Line, l.Col)
}

var _ error = Error{}

type Error struct {
	Range
	Message string
	Wrapped error
}

func (e Error) Error() string {
	var s strings.Builder
	if e.Wrapped != nil {
		s.WriteString(e.Wrapped.Error())
		s.WriteString("\n")
	} else {
		s.WriteString(e.Text)
	}
	if len(e.Path) > 0 {
		s.WriteString(e.Path)
		s.WriteString(": ")
	}
	loc := e.Location()
	s.WriteString(loc.String())
	s.WriteString(" ")
	s.WriteString(e.Message)
	return s.String()
}
