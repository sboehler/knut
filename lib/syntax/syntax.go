package syntax

import (
	"text/scanner"

	"github.com/sboehler/knut/lib/syntax/directives"
	"github.com/sboehler/knut/lib/syntax/parser"
	"github.com/sboehler/knut/lib/syntax/printer"
)

type Commodity = directives.Commodity

type Account = directives.Account

type Date = directives.Date

type Decimal = directives.Decimal

type QuotedString = directives.QuotedString

type Booking = directives.Booking

type Performance = directives.Performance

type Interval = directives.Interval

type Directive = directives.Directive

type File = directives.File

type Accrual = directives.Accrual

type Addons = directives.Addons

type Transaction = directives.Transaction

type Open = directives.Open

type Close = directives.Close

type Assertion = directives.Assertion

type Price = directives.Price

type Include = directives.Include

type Range = directives.Range

func SetRange[T any, P interface {
	*T
	SetRange(Range)
}](t P, r Range) T {
	t.SetRange(r)
	return *t
}

type Location = directives.Location

var _ error = Error{}

type Error = directives.Error

type Parser = parser.Parser

type Scanner = scanner.Scanner

type Printer = printer.Printer

var Parse = parser.Parse

var NewParser = parser.New
