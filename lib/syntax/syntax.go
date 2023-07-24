package syntax

import (
	"fmt"
	"strings"
)

type Commodity struct{ Range }

type Account struct{ Range }

type AccountMacro struct{ Range }

type Date struct{ Range }

type Decimal struct{ Range }

type QuotedString struct{ Range }

type Booking struct {
	Range
	Credit, Debit           Account
	CreditMacro, DebitMacro AccountMacro
	Amount                  Decimal
	Commodity               Commodity
}

type Performance struct {
	Range
	Targets []Commodity
}

type Interval Range

type Accrual struct {
	Range
	Interval   Interval
	Start, End Date
}

type Addons struct {
	Performance *Performance
	Accrual     *Accrual
}

type Transaction struct {
	Range
	Date        Date
	Description QuotedString
	Bookings    []Booking
	Accrual     *Accrual
	Performance *Performance
}

type Range struct {
	Start, End int
	Path, Text string
}

func (r *Range) SetEnd(end int) {
	r.End = end
}

func (r Range) Empty() bool {
	return r.Start == r.End
}

func (r Range) Location() [2]Location {
	loc := Location{Line: 1, Col: 1}
	var res [2]Location
	for pos, ch := range r.Text {
		if pos == r.Start {
			res[0] = loc
		} else if pos == r.End {
			res[1] = loc
			return res
		}
		if ch == '\n' {
			loc.Line++
			loc.Col = 1
		} else {
			loc.Col++
		}
	}
	return res
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
	// var s strings.Builder
	// if len(e.Path) > 0 {
	// 	s.WriteString(e.Path)
	// 	s.WriteString(": ")
	// }
	// s.WriteString(e.Location()[1].String())
	// s.WriteString(" ")
	// s.WriteString(e.Message)
	return e.Message
}
