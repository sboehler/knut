package report2

import (
	"sync"

	"github.com/sboehler/knut/lib/common/amounts"
	"github.com/sboehler/knut/lib/common/compare"
	"github.com/sboehler/knut/lib/common/dict"
	"github.com/sboehler/knut/lib/journal"
	"github.com/shopspring/decimal"
)

type Report struct {
	Context journal.Context
	AL, EIE *Node
}

func NewReport(jctx journal.Context) *Report {
	return &Report{
		Context: jctx,
		AL:      newNode(nil),
		EIE:     newNode(nil),
	}
}

func (r *Report) Insert(k amounts.Key, v decimal.Decimal) {
	ancestors := r.Context.Accounts().Ancestors(k.Account)
	if k.Account.Type() == journal.ASSETS || k.Account.Type() == journal.LIABILITIES {
		r.AL.Insert(ancestors, k, v)
	} else {
		r.EIE.Insert(ancestors, k, v)
	}
}

func (r *Report) ComputeWeights() {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		r.AL.computeWeights()
		wg.Done()
	}()
	go func() {
		r.EIE.computeWeights()
		wg.Done()
	}()
	wg.Wait()
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

func (n *Node) Insert(as []*journal.Account, k amounts.Key, v decimal.Decimal) {
	if len(as) == 0 {
		n.Amounts.Add(k, v)
	} else {
		head, tail := as[0], as[1:]
		dict.
			GetDefault(n.children, head, func() *Node { return newNode(head) }).
			Insert(tail, k, v)
	}
}

func (n *Node) Children() []*Node {
	return dict.SortedValues(n.children, compareNodes)
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
	var wg sync.WaitGroup
	wg.Add(len(n.children))
	for _, sn := range n.children {
		go func(sn *Node) {
			sn.computeWeights()
			wg.Done()
		}(sn)
	}
	n.weight = 0
	keysWithVal := func(k amounts.Key) bool { return k.Valuation != nil }
	w := n.Amounts.SumOver(keysWithVal)
	f, _ := w.Float64()
	n.weight += f
	wg.Wait()
	for _, sn := range n.children {
		n.weight += sn.weight
	}
}
