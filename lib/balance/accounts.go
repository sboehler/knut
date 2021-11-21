package balance

import (
	"fmt"

	"github.com/sboehler/knut/lib/ledger"
)

// Accounts keeps track of open accounts.
type Accounts map[*ledger.Account]bool

// Open opens an account.
func (oa Accounts) Open(a *ledger.Account) error {
	if oa[a] {
		return fmt.Errorf("account %v is already open", a)
	}
	oa[a] = true
	return nil
}

// Close closes an account.
func (oa Accounts) Close(a *ledger.Account) error {
	if !oa[a] {
		return fmt.Errorf("account %v is already closed", a)
	}
	delete(oa, a)
	return nil
}

// IsOpen returns whether an account is open.
func (oa Accounts) IsOpen(a *ledger.Account) bool {
	if oa[a] {
		return true
	}
	return a.Type() == ledger.EQUITY
}

// Copy copies accounts.
func (oa Accounts) Copy() Accounts {
	var res = make(map[*ledger.Account]bool, len(oa))
	for a := range oa {
		res[a] = true
	}
	return res
}
