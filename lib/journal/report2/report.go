package report2

import (
	"github.com/sboehler/knut/lib/common/amounts"
	"github.com/sboehler/knut/lib/journal"
	"github.com/shopspring/decimal"
)

type Report struct {
	Context journal.Context
	AL, EIE Section
}

func NewReport(jctx journal.Context) *Report {
	return &Report{
		Context: jctx,
		AL: Section{
			Nodes: make(map[journal.AccountType]*Node),
		},
		EIE: Section{
			Nodes: make(map[journal.AccountType]*Node),
		},
	}
}

func (r *Report) Insert(k amounts.Key, v decimal.Decimal) {
	if k.Account.Type() == journal.ASSETS || k.Account.Type() == journal.LIABILITIES {
		r.AL.Insert(r.Context, k, v)
	} else {
		r.EIE.Insert(r.Context, k, v)
	}
}

type Section struct {
	Nodes map[journal.AccountType]*Node

	// Date [* Commodity]
	Totals map[amounts.Key]decimal.Decimal
}

func (s *Section) Insert(jctx journal.Context, k amounts.Key, v decimal.Decimal) {
	ancestors := jctx.Accounts().Ancestors(k.Account)
	root := ancestors[0]
	n, ok := s.Nodes[root.Type()]
	if !ok {
		n = newNode(root)
		s.Nodes[root.Type()] = n
	}
	n.Insert(ancestors, k, v)
}

type Node struct {
	Account  *journal.Account
	Children map[*journal.Account]*Node
	Amounts  amounts.Amounts
}

func newNode(a *journal.Account) *Node {
	return &Node{
		Account:  a,
		Children: make(map[*journal.Account]*Node),
		Amounts:  make(amounts.Amounts),
	}

}

func (n *Node) Insert(as []*journal.Account, k amounts.Key, v decimal.Decimal) {
	if len(as) == 0 {
		n.Amounts.Add(k, v)
		return
	}
	head, tail := as[0], as[1:]
	ch, ok := n.Children[head]
	if !ok {
		ch = newNode(head)
		n.Children[head] = ch
	}
	ch.Insert(tail, k, v)
}
