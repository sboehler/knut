package val

import (
	"time"

	"github.com/sboehler/knut/lib/balance/prices"
	"github.com/sboehler/knut/lib/common/amounts"
	"github.com/sboehler/knut/lib/journal/ast"
	"github.com/sboehler/knut/lib/journal/past"
)

// Day is a day with valuated transactions and positions.
type Day struct {
	Day          *past.Day
	Date         time.Time
	Prices       prices.NormalizedPrices
	Transactions []*ast.Transaction
	Values       amounts.Amounts
}
