package repo

import "github.com/sboehler/knut/pkg/repo/db"

// AccountType is the type of an account.
type AccountType int

const (
	// ASSETS represents an asset account.
	ASSETS AccountType = iota
	// LIABILITIES represents a liability account.
	LIABILITIES
	// EQUITY represents an equity account.
	EQUITY
	// INCOME represents an income account.
	INCOME
	// EXPENSES represents an expenses account.
	EXPENSES
)

// Account is an account.
type Account struct {
	AccountID   uint64
	AccountType AccountType
	Name        string
}

func (a Account) ID() uint64 {
	return a.AccountID
}

func (a *Account) SetID(id uint64) {
	a.AccountID = id
}

type Commodity struct {
	CommodityID uint64
}

func (a Commodity) ID() uint64 {
	return a.CommodityID
}

func (a *Commodity) SetID(id uint64) {
	a.CommodityID = id
}

var (
	AccountTable   = db.Table[Account, *Account]{Name: "accounts", TableID: 0x1}
	CommodityTable = db.Table[Commodity, *Commodity]{Name: "commodities", TableID: 0x2}
)
