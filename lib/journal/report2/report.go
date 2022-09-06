package report2

import (
	"github.com/sboehler/knut/lib/common/amounts"
	"github.com/sboehler/knut/lib/common/maputils"
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
	maputils.
		GetDefault(s.Nodes, root.Type(), func() *Node { return newNode(root) }).
		Insert(ancestors, k, v)
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
	} else {
		head, tail := as[0], as[1:]
		maputils.
			GetDefault(n.Children, head, func() *Node { return newNode(head) }).
			Insert(tail, k, v)
	}
}
