package balance

import (
	"github.com/sboehler/knut/lib/amounts"
	"github.com/sboehler/knut/lib/common/compare"
	"github.com/sboehler/knut/lib/common/date"
	"github.com/sboehler/knut/lib/common/mapper"
	"github.com/sboehler/knut/lib/common/multimap"
	"github.com/sboehler/knut/lib/model"
	"github.com/sboehler/knut/lib/model/account"
	"github.com/shopspring/decimal"
)

type Report struct {
	Registry  *model.Registry
	AL, EIE   *multimap.Node[Value]
	partition date.Partition
}

type Value struct {
	Account *model.Account
	Amounts amounts.Amounts
	Weight  decimal.Decimal
}

type Node = multimap.Node[Value]

func NewReport(reg *model.Registry, part date.Partition) *Report {
	return &Report{
		Registry:  reg,
		AL:        multimap.New[Value](""),
		EIE:       multimap.New[Value](""),
		partition: part,
	}
}

func (r *Report) Insert(k amounts.Key, v decimal.Decimal) {
	if k.Account == nil {
		return
	}
	var n *Node
	if k.Account.IsAL() {
		n = r.AL.GetOrCreate(k.Account.Segments())
	} else {
		n = r.EIE.GetOrCreate(k.Account.Segments())
	}
	if n.Value.Account == nil {
		n.Value.Account = k.Account
		n.Value.Amounts = make(amounts.Amounts)
	}
	n.Value.Amounts.Add(k, v)
}

func (r *Report) SortAlpha() {
	f := func(n1, n2 *Node) compare.Order {
		if n1.Value.Account.Level() == 1 && n2.Value.Account.Level() == 1 {
			return compare.Ordered(n1.Value.Account.Type(), n2.Value.Account.Type())
		}
		return multimap.SortAlpha(n1, n2)
	}
	r.AL.Sort(f)
	r.EIE.Sort(f)
}

func (r *Report) SortWeighted() {
	computeWeights := func(n *Node) {
		w := n.Value.Amounts.SumOver(func(k amounts.Key) bool {
			return k.Valuation != nil
		}).Abs().Neg()
		for _, ch := range n.Children {
			w = w.Add(ch.Value.Weight)
		}
		n.Value.Weight = w
	}
	r.AL.PostOrder(computeWeights)
	r.EIE.PostOrder(computeWeights)
	f := func(n1, n2 *Node) compare.Order {
		if n1.Value.Account.Level() == 1 && n2.Value.Account.Level() == 1 {
			return compare.Ordered(n1.Value.Account.Type(), n2.Value.Account.Type())
		}
		return compare.Decimal(n1.Value.Weight, n2.Value.Weight)
	}
	r.AL.Sort(f)
	r.EIE.Sort(f)
}

func (r *Report) SetAccounts() {
	setAccounts(r.Registry.Accounts(), r.AL)
	setAccounts(r.Registry.Accounts(), r.EIE)
}

func setAccounts(reg *account.Registry, n *Node) {
	var acc *account.Account
	for _, ch := range n.Children {
		setAccounts(reg, ch)
		if acc == nil {
			acc = reg.Parent(ch.Value.Account)
		}
	}
	if n.Value.Account == nil {
		n.Value.Account = acc
	}
}

func (r *Report) Totals(m mapper.Mapper[amounts.Key]) (amounts.Amounts, amounts.Amounts) {
	al, eie := make(amounts.Amounts), make(amounts.Amounts)
	r.AL.PostOrder(func(n *Node) {
		n.Value.Amounts.SumIntoBy(al, nil, m)
	})
	r.EIE.PostOrder(func(n *Node) {
		n.Value.Amounts.SumIntoBy(eie, nil, m)
	})
	return al, eie
}
