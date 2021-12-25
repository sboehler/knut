package val

import (
	"time"

	"github.com/sboehler/knut/lib/balance/prices"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
	"github.com/sboehler/knut/lib/journal/past"
	"github.com/shopspring/decimal"
)

// Day is a day with valuated transactions and positions.
type Day struct {
	Day          *past.Day
	Date         time.Time
	Prices       prices.NormalizedPrices
	Transactions []*Transaction
	Values       past.Amounts
}

// Transaction represents a valuated transaction.
type Transaction struct {
	Source   *ast.Transaction
	Postings []Posting
}

// Posting is a valuated posting.
type Posting struct {
	Source        *ast.Posting
	Credit, Debit *journal.Account
	Value         decimal.Decimal
	Commodity     *journal.Commodity
}
