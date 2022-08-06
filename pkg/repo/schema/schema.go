package schema

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

type AccountID int64

// Account is an account.
type Account struct {
	AccountID   AccountID
	AccountType AccountType
	Name        string
}

func (a Account) ID() uint64 {
	return uint64(a.AccountID)
}

func (a *Account) SetID(id uint64) {
	a.AccountID = AccountID(id)
}
