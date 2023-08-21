package weights

import (
	"time"

	"github.com/sboehler/knut/lib/common/date"
	"github.com/sboehler/knut/lib/common/multimap"
	"github.com/sboehler/knut/lib/common/set"
	"github.com/sboehler/knut/lib/common/table"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/performance"
	"github.com/sboehler/knut/lib/model"
)

type Report struct {
	Registry    *model.Registry
	universe    performance.Universe
	partition   date.Partition
	dates       set.Set[time.Time]
	commodities set.Set[*model.Commodity]
	weights     *Node
}
type Node = multimap.Node[Value]

func NewReport(reg *model.Registry, ds date.Partition, universe performance.Universe) *Report {
	endDates := set.New[time.Time]()
	for _, d := range ds.EndDates() {
		endDates.Add(d)
	}
	return &Report{
		Registry:    reg,
		universe:    universe,
		partition:   ds,
		dates:       endDates,
		commodities: set.New[*model.Commodity](),

		weights: multimap.New[Value](""),
	}
}

type Value struct {
	Commodity *model.Commodity
	Weights   map[time.Time]float64
}

func (r *Report) Add(d *journal.Day) error {
	if !r.dates.Has(d.Date) {
		return nil
	}
	var total float64
	for _, v := range d.Performance.V1 {
		total += v
	}
	for com, v := range d.Performance.V1 {
		r.commodities.Add(com)
		ss := r.universe.Locate(com)
		n := r.weights.GetOrCreate(ss)
		if n.Value.Weights == nil {
			n.Value.Weights = make(map[time.Time]float64)
			n.Value.Commodity = com
		}
		n.Value.Weights[d.Date] = v / total
	}
	return nil
}

func (r *Report) PropagateWeights() {
	r.weights.PostOrder(func(n *Node) {
		if n.Value.Weights == nil {
			n.Value.Weights = make(map[time.Time]float64)
		}
		for _, ch := range n.Children {
			for _, date := range r.partition.EndDates() {
				n.Value.Weights[date] += ch.Value.Weights[date]
			}
		}
	})
}

type Renderer struct {
	OmitCommodities bool
	table           *table.Table
	report          *Report
}

func (rn *Renderer) Render(rep *Report) *table.Table {
	rep.weights.Sort(multimap.SortAlpha)
	rep.PropagateWeights()

	rn.table = table.New(1, rep.partition.Size())
	rn.report = rep

	rn.table.AddSeparatorRow()
	rn.renderHeader()
	rn.table.AddSeparatorRow()

	for _, n := range rep.weights.Sorted {
		rn.renderNode(n, 0)
	}
	rn.table.AddSeparatorRow()

	return rn.table
}

func (rn *Renderer) renderHeader() {
	row := rn.table.AddRow()
	row.AddText("Commodity", table.Center)
	for _, d := range rn.report.partition.EndDates() {
		row.AddText(d.Format("2006-01-02"), table.Center)
	}
}

func (rn *Renderer) renderNode(n *Node, indent int) {
	row := rn.table.AddRow()
	row.AddIndented(n.Segment, indent)
	for _, date := range rn.report.partition.EndDates() {
		if w, ok := n.Value.Weights[date]; ok && w != 0 {
			row.AddPercent(w)
		} else {
			row.AddEmpty()
		}
	}
	for _, ch := range n.Sorted {
		if ch.Value.Commodity == nil || !rn.OmitCommodities {
			rn.renderNode(ch, indent+2)
		}
	}
}
