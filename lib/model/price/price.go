package price

import (
	"time"

	"github.com/sboehler/knut/lib/model/commodity"
	"github.com/sboehler/knut/lib/model/registry"
	"github.com/sboehler/knut/lib/syntax"
	"github.com/shopspring/decimal"
)

// Price represents a price command.
type Price struct {
	Src       *syntax.Price
	Date      time.Time
	Commodity *commodity.Commodity
	Price     decimal.Decimal
	Target    *commodity.Commodity
}

func Create(reg *registry.Registry, p *syntax.Price) (*Price, error) {
	date, err := p.Date.Parse()
	if err != nil {
		return nil, err
	}
	com, err := reg.Commodities().Create(p.Commodity)
	if err != nil {
		return nil, err
	}
	pr, err := p.Price.Parse()
	if err != nil {
		return nil, err
	}
	tgt, err := reg.Commodities().Create(p.Target)
	if err != nil {
		return nil, err
	}
	return &Price{
		Src:       p,
		Date:      date,
		Commodity: com,
		Price:     pr,
		Target:    tgt,
	}, nil
}
