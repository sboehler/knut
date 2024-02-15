package weights

import (
	"strings"
	"time"

	"github.com/sboehler/knut/lib/common/compare"
	"github.com/sboehler/knut/lib/common/date"
	"github.com/sboehler/knut/lib/common/multimap"
	"github.com/sboehler/knut/lib/common/set"
	"github.com/sboehler/knut/lib/common/table"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/performance"
	"github.com/sboehler/knut/lib/model/account"
)

type Query struct {
	Partition date.Partition
	Universe  performance.Universe
	Mapping   account.Mapping
}

func (q Query) Execute(j *journal.Builder, r *Report) *journal.Processor {
	days := set.FromSlice(j.Days(q.Partition.EndDates()))
	return &journal.Processor{
		DayEnd: func(d *journal.Day) error {
			if !days.Has(d) {
				return nil
			}
			var total float64
			for _, v := range d.Performance.V1 {
				total += v
			}
			for com, v := range d.Performance.V1 {
				ss := q.Universe.Locate(com)
				level, suffix, ok := q.Mapping.Level(strings.Join(ss, ":"))
				if ok && level < len(ss)-suffix {
					ss = append(ss[:level], ss[len(ss)-suffix:]...)
				}
				r.Add(ss, d.Date, v/total)
			}
			return nil
		},
	}
}

type Node = multimap.Node[Value]

type Value struct {
	Leaf    bool
	Weights map[time.Time]float64
	Weight  float64
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
	n.Value.Weights[date] += w
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

func (r *Report) SortWeighted() {
	r.weights.PostOrder(func(n *Node) {
		var total float64
		for _, w := range n.Value.Weights {
			total += w
		}
		n.Value.Weight = -total
	})
	r.weights.Sort(func(n1, n2 *Node) compare.Order {
		return compare.Ordered(n1.Value.Weight, n2.Value.Weight)
	})
}

type Renderer struct {
	SortAlphabetically bool

	table  *table.Table
	report *Report
	dates  []time.Time
}

func (rn *Renderer) Render(rep *Report) *table.Table {
	rep.PropagateWeights()
	if rn.SortAlphabetically {
		rep.weights.Sort(multimap.SortAlpha)
	} else {
		rep.SortWeighted()
	}

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
