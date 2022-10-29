package report

import (
	"time"

	"github.com/sboehler/knut/lib/common/amounts"
	"github.com/sboehler/knut/lib/common/compare"
	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/common/dict"
	"github.com/sboehler/knut/lib/common/mapper"
	"github.com/sboehler/knut/lib/journal"
	"github.com/shopspring/decimal"
)

type Report struct {
	Context journal.Context
	AL, EIE *Node
	cache   nodeCache
}

type nodeCache map[*journal.Account]*Node

func NewReport(jctx journal.Context) *Report {
	return &Report{
		Context: jctx,
		AL:      newNode(nil),
		EIE:     newNode(nil),
		cache:   make(nodeCache),
	}
}

func (r *Report) Insert(k amounts.Key, v decimal.Decimal) {
	if k.Account == nil {
		return
	}
	n, ok := r.cache[k.Account]
	if !ok {
		ancestors := r.Context.Accounts().Ancestors(k.Account)
		if k.Account.IsAL() {
			n = r.AL.Leaf(ancestors)
		} else {
			n = r.EIE.Leaf(ancestors)
		}
		r.cache[k.Account] = n
	}
	n.Insert(k, v)
}

func (r *Report) ComputeWeights() {
	cpr.Parallel(
		func() { r.AL.computeWeights() },
		func() { r.EIE.computeWeights() },
	)()
}

func (r *Report) Totals() (amounts.Amounts, amounts.Amounts) {
	res1, res2 := make(amounts.Amounts), make(amounts.Amounts)
	cpr.Parallel(
		func() { r.AL.computeTotals(res1) },
		func() { r.EIE.computeTotals(res2) },
	)()
	return res1, res2
}

type Node struct {
	Account  *journal.Account
	children map[*journal.Account]*Node
	Amounts  amounts.Amounts

	weight float64
}

func newNode(a *journal.Account) *Node {
	return &Node{
		Account:  a,
		children: make(map[*journal.Account]*Node),
		Amounts:  make(amounts.Amounts),
	}
}

func (n *Node) Insert(k amounts.Key, v decimal.Decimal) {
	n.Amounts.Add(k, v)
}

func (n *Node) Leaf(as []*journal.Account) *Node {
	if len(as) == 0 {
		return n
	}
	head, tail := as[0], as[1:]
	return dict.
		GetDefault(n.children, head, func() *Node { return newNode(head) }).
		Leaf(tail)
}

func (n *Node) Children() []*Node {
	return dict.SortedValues(n.children, compareNodes)
}

func (n *Node) Segment() string {
	return n.Account.Segment()
}

func compareNodes(n1, n2 *Node) compare.Order {
	if n1.Account.Type() != n2.Account.Type() {
		return compare.Ordered(n1.Account.Type(), n2.Account.Type())
	}
	if n1.weight != n2.weight {
		return compare.Ordered(n1.weight, n2.weight)
	}
	return compare.Ordered(n1.Account.Name(), n2.Account.Name())
}

func (n *Node) computeWeights() {
	wait := cpr.ForAll(n.Children(), func(sn *Node) {
		sn.computeWeights()
	})
	n.weight = 0
	keysWithVal := func(k amounts.Key) bool { return k.Valuation != nil }
	w := n.Amounts.SumOver(keysWithVal)
	f, _ := w.Abs().Float64()
	n.weight -= f
	wait()
	for _, sn := range n.children {
		n.weight += sn.weight
	}
}

func (n *Node) computeTotals(m amounts.Amounts) {
	for _, ch := range n.children {
		ch.computeTotals(m)
	}
	n.Amounts.SumIntoBy(m, nil, amounts.KeyMapper{
		Date:      mapper.Identity[time.Time],
		Commodity: mapper.Identity[*journal.Commodity],
	}.Build())
}
