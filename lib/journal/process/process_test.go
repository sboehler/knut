package process

import (
	"time"

	"github.com/sboehler/knut/lib/common/date"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
	"github.com/shopspring/decimal"
)

type TestData struct {
	date1, date2, date3    time.Time
	account1, account2     *journal.Account
	commodity1, commodity2 *journal.Commodity
	open1, open2           *ast.Open
	price1                 *ast.Price
	trx1                   *ast.Transaction
}

func newTestData(jctx journal.Context) TestData {
	var (
		date1      = date.Date(2022, 1, 4)
		date2      = date.Date(2022, 1, 5)
		date3      = date.Date(2022, 1, 6)
		account1   = jctx.Account("Assets:Account")
		account2   = jctx.Account("Assets:Other")
		commodity1 = jctx.Commodity("COM")
		commodity2 = jctx.Commodity("TGT")
		price1     = &ast.Price{
			Date:      date1,
			Commodity: commodity1,
			Target:    commodity2,
			Price:     decimal.NewFromInt(4),
		}
		open1 = &ast.Open{Date: date2, Account: account1}
		open2 = &ast.Open{Date: date2, Account: account2}
		trx1  = &ast.Transaction{
			Date:        date2,
			Description: "foo",
			Postings: []ast.Posting{
				ast.NewPosting(account1, account2, commodity1, decimal.NewFromInt(10)),
			},
		}
	)
	return TestData{
		date1, date2, date3,
		account1, account2,
		commodity1, commodity2,
		open1, open2,
		price1,
		trx1,
	}
}
