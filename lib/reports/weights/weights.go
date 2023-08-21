package weights

import (
	"time"

	"github.com/sboehler/knut/lib/common/compare"
	"github.com/sboehler/knut/lib/common/date"
	"github.com/sboehler/knut/lib/common/multimap"
	"github.com/sboehler/knut/lib/common/set"
	"github.com/sboehler/knut/lib/common/table"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/performance"
)

type Query struct {
	OmitCommodities bool
	Partition       date.Partition
	Universe        performance.Universe
}

func (q Query) Execute(j *journal.Journal, r *Report) journal.DayFn {
	days := set.New[*journal.Day]()
	for _, d := range q.Partition.EndDates() {
		days.Add(j.Day(d))
	}
	return func(d *journal.Day) error {
		if !days.Has(d) {
			return nil
		}
		var total float64
		for _, v := range d.Performance.V1 {
			total += v
		}
		for com, v := range d.Performance.V1 {
			ss := q.Universe.Locate(com)
			if q.OmitCommodities {
				ss = ss[:len(ss)-1]
			}
			r.Add(ss, d.Date, v/total)
		}
		return nil
	}
}

type Node = multimap.Node[Value]

type Value struct {
	Leaf    bool
	Weights map[time.Time]float64
}

type Report struct {
	dates   set.Set[time.Time]
	weights *Node
}

func NewReport() *Report {
	return &Report{
		dates:   set.New[time.Time](),
		weights: multimap.New[Value](""),
	}
}

func (r *Report) Add(ss []string, date time.Time, w float64) error {
	n := r.weights.GetOrCreate(ss)
	if n.Value.Weights == nil {
		n.Value.Weights = make(map[time.Time]float64)
		n.Value.Leaf = true
	}
	n.Value.Weights[date] = w
	r.dates.Add(date)
	return nil
}

func (r *Report) PropagateWeights() {
	r.weights.PostOrder(func(n *Node) {
		if n.Value.Weights == nil {
			n.Value.Weights = make(map[time.Time]float64)
		}
		for _, ch := range n.Children {
			for date, w := range ch.Value.Weights {
				n.Value.Weights[date] += w
			}
		}
	})
}

type Renderer struct {
	table  *table.Table
	report *Report
	dates  []time.Time
}

func (rn *Renderer) Render(rep *Report) *table.Table {
	rep.weights.Sort(multimap.SortAlpha)
	rep.PropagateWeights()

	rn.dates = rep.dates.Sorted(compare.Time)
	rn.table = table.New(1, len(rn.dates))
	rn.report = rep

	rn.table.AddSeparatorRow()
	rn.renderHeader()
	rn.table.AddSeparatorRow()

	for _, node := range rep.weights.Sorted {
		rn.renderNode(node, 0)
	}
	rn.table.AddSeparatorRow()

	return rn.table
}

func (rn *Renderer) renderHeader() {
	row := rn.table.AddRow()
	row.AddText("Commodity", table.Center)
	for _, date := range rn.dates {
		row.AddText(date.Format("2006-01-02"), table.Center)
	}
}

func (rn *Renderer) renderNode(n *Node, indent int) {
	row := rn.table.AddRow()
	row.AddIndented(n.Segment, indent)
	for _, date := range rn.dates {
		if weight, ok := n.Value.Weights[date]; ok && weight != 0 {
			row.AddPercent(weight)
		} else {
			row.AddEmpty()
		}
	}
	for _, child := range n.Sorted {
		rn.renderNode(child, indent+2)
	}
}
