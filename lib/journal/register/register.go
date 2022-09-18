package register

import (
	"time"

	"github.com/sboehler/knut/lib/common/amounts"
	"github.com/sboehler/knut/lib/common/compare"
	"github.com/sboehler/knut/lib/common/dict"
	"github.com/sboehler/knut/lib/common/table"
	"github.com/sboehler/knut/lib/journal"
	"github.com/shopspring/decimal"
)

type Report struct {
	Context journal.Context

	nodes map[time.Time]*Node
}

type Node struct {
	Date    time.Time
	Amounts amounts.Amounts
}

func NewReport(jctx journal.Context) *Report {
	return &Report{
		nodes: make(map[time.Time]*Node),
	}
}

func newNode(d time.Time) *Node {
	return &Node{
		Date:    d,
		Amounts: make(amounts.Amounts),
	}
}

func (r *Report) Insert(k amounts.Key, v decimal.Decimal) {
	n := dict.GetDefault(r.nodes, k.Date, func() *Node { return newNode(k.Date) })
	n.Amounts.Add(k, v)
}

type Renderer struct {
	ShowCommodities    bool
	SortAlphabetically bool
}

func (rn *Renderer) Render(r *Report) *table.Table {
	var tbl *table.Table
	if rn.ShowCommodities {
		tbl = table.New(1, 1, 1, 1)
	} else {
		tbl = table.New(1, 1, 1)
	}
	tbl.AddSeparatorRow()
	header := tbl.AddRow().
		AddText("Date", table.Center).
		AddText("Account", table.Center)
	if rn.ShowCommodities {
		header.AddText("Comm", table.Center)
	}
	header.AddText("Amount", table.Center)
	tbl.AddSeparatorRow()

	dates := dict.SortedKeys(r.nodes, compare.Time)
	for _, d := range dates {
		n := r.nodes[d]
		rn.renderNode(tbl, n)
	}
	tbl.AddSeparatorRow()
	return tbl
}

func (rn *Renderer) renderNode(tbl *table.Table, n *Node) {
	idx := n.Amounts.Index(func(k1, k2 amounts.Key) compare.Order {
		if c := journal.CompareAccounts(k1.Other, k2.Other); c != compare.Equal {
			return c
		}
		return journal.CompareCommodities(k1.Commodity, k2.Commodity)
	})
	for i, k := range idx {
		row := tbl.AddRow()
		if i == 0 {
			row.AddText(n.Date.Format("2006-01-02"), table.Left)
		} else {
			row.AddEmpty()
		}
		row.AddText(k.Other.Name(), table.Left)
		if rn.ShowCommodities {
			row.AddText(k.Commodity.Name(), table.Left)
		}
		row.AddNumber(n.Amounts[k])
	}
	tbl.AddEmptyRow()
}
