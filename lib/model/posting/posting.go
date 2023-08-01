package posting

import (
	"github.com/sboehler/knut/lib/common/compare"
	"github.com/sboehler/knut/lib/model/account"
	"github.com/sboehler/knut/lib/model/commodity"
	"github.com/sboehler/knut/lib/model/registry"
	"github.com/sboehler/knut/lib/syntax"
	"github.com/shopspring/decimal"
)

// Posting represents a posting.
type Posting struct {
	Src            *syntax.Booking
	Amount, Value  decimal.Decimal
	Account, Other *account.Account
	Commodity      *commodity.Commodity
}

type Builder struct {
	Src           *syntax.Booking
	Amount, Value decimal.Decimal
	Credit, Debit *account.Account
	Commodity     *commodity.Commodity
}

func (pb Builder) Build() []*Posting {
	if pb.Amount.IsNegative() || pb.Amount.IsZero() && pb.Value.IsNegative() {
		pb.Credit, pb.Debit, pb.Amount, pb.Value = pb.Debit, pb.Credit, pb.Amount.Neg(), pb.Value.Neg()
	}
	return []*Posting{
		{
			Src:       pb.Src,
			Account:   pb.Credit,
			Other:     pb.Debit,
			Commodity: pb.Commodity,
			Amount:    pb.Amount.Neg(),
			Value:     pb.Value.Neg(),
		},
		{
			Src:       pb.Src,
			Account:   pb.Debit,
			Other:     pb.Credit,
			Commodity: pb.Commodity,
			Amount:    pb.Amount,
			Value:     pb.Value,
		},
	}
}

type Builders []Builder

func (pbs Builders) Build() []*Posting {
	res := make([]*Posting, 0, 2*len(pbs))
	for _, pb := range pbs {
		res = append(res, pb.Build()...)
	}
	return res
}

func Compare(p, p2 *Posting) compare.Order {
	if o := account.Compare(p.Account, p2.Account); o != compare.Equal {
		return o
	}
	if o := account.Compare(p.Other, p2.Other); o != compare.Equal {
		return o
	}
	if o := compare.Decimal(p.Amount, p2.Amount); o != compare.Equal {
		return o
	}
	if o := compare.Decimal(p.Value, p2.Value); o != compare.Equal {
		return o
	}
	return compare.Ordered(p.Commodity.Name(), p2.Commodity.Name())
}

func Create(reg *registry.Registry, bs []syntax.Booking) ([]*Posting, error) {
	var builder Builders
	for i, b := range bs {
		credit, err := reg.Accounts().Create(b.Credit)
		if err != nil {
			return nil, err
		}
		debit, err := reg.Accounts().Create(b.Debit)
		if err != nil {
			return nil, err
		}
		amount, err := decimal.NewFromString(b.Amount.Extract())
		if err != nil {
			return nil, syntax.Error{Range: b.Amount.Range, Message: "parsing amount", Wrapped: err}
		}
		commodity, err := reg.Commodities().Create(b.Commodity)
		if err != nil {
			return nil, err
		}
		builder = append(builder, Builder{
			Src:       &bs[i],
			Credit:    credit,
			Debit:     debit,
			Amount:    amount,
			Commodity: commodity,
		})
	}
	return builder.Build(), nil
}
