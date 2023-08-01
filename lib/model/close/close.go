package close

import (
	"time"

	"github.com/sboehler/knut/lib/model/account"
	"github.com/sboehler/knut/lib/model/registry"
	"github.com/sboehler/knut/lib/syntax"
)

// Open represents an open command.
type Close struct {
	Src     *syntax.Close
	Date    time.Time
	Account *account.Account
}

func Create(reg *registry.Registry, c *syntax.Close) (*Close, error) {
	account, err := reg.Accounts().Create(c.Account)
	if err != nil {
		return nil, err
	}
	date, err := c.Date.Parse()
	if err != nil {
		return nil, err
	}
	return &Close{
		Src:     c,
		Date:    date,
		Account: account,
	}, nil
}
