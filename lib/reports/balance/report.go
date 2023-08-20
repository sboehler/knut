package balance

import (
	"github.com/sboehler/knut/lib/amounts"
	"github.com/sboehler/knut/lib/common/date"
	"github.com/sboehler/knut/lib/common/mapper"
	"github.com/sboehler/knut/lib/common/multimap"
	"github.com/sboehler/knut/lib/model"
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
		n = r.AL.GetOrCreate(k.Account.Split())
	} else {
		n = r.EIE.GetOrCreate(k.Account.Split())
	}
	if n.Value.Account == nil {
		n.Value.Account = k.Account
		n.Value.Amounts = make(amounts.Amounts)
	}
	n.Value.Amounts.Add(k, v)
}

func (r *Report) ComputeWeights() {
	f := func(n *Node) {
		n.Weight = 0
		keysWithVal := func(k amounts.Key) bool { return k.Valuation != nil }
		w := n.Value.Amounts.SumOver(keysWithVal)
		f, _ := w.Abs().Float64()
		n.Weight -= f
		for _, sn := range n.Children() {
			n.Weight += sn.Weight
		}
	}
	r.AL.PostOrder(f)
	r.EIE.PostOrder(f)
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
