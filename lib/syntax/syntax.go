package syntax

import "github.com/sboehler/knut/lib/syntax/scanner"

type Pos = scanner.Range

type Commodity Pos

type Account Pos

type AccountMacro Pos

type Date Pos

type Decimal Pos

type QuotedString Pos

type Booking struct {
	Pos
	Credit, Debit           Account
	CreditMacro, DebitMacro AccountMacro
	Amount                  Decimal
	Commodity               Commodity
}

type Performance struct {
	Pos
	Targets []Commodity
}

type Interval Pos

type Accrual struct {
	Pos
	Interval   Interval
	Start, End Date
}

type Addons struct {
	Performance *Performance
	Accrual     *Accrual
}

type Transaction struct {
	Pos
	Date        Date
	Description QuotedString
	Bookings    []Booking
	Accrual     *Accrual
	Performance *Performance
}
