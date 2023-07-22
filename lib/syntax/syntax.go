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
