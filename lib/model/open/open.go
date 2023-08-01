package open

import (
	"time"

	"github.com/sboehler/knut/lib/model/account"
	"github.com/sboehler/knut/lib/model/registry"
	"github.com/sboehler/knut/lib/syntax"
)

// Open represents an open command.
type Open struct {
	Src     *syntax.Open
	Date    time.Time
	Account *account.Account
}

func Create(reg *registry.Registry, o *syntax.Open) (*Open, error) {
	account, err := reg.Accounts().Create(o.Account)
	if err != nil {
		return nil, err
	}
	date, err := o.Date.Parse()
	if err != nil {
		return nil, err
	}
	return &Open{
		Src:     o,
		Date:    date,
		Account: account,
	}, nil
}
