package weights

import (
	"time"

	"github.com/sboehler/knut/lib/common/date"
	"github.com/sboehler/knut/lib/common/set"
	"github.com/sboehler/knut/lib/common/table"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/model"
	"github.com/sboehler/knut/lib/model/commodity"
)

type Report struct {
	Registry    *model.Registry
	part        date.Partition
	dates       set.Set[time.Time]
	commodities set.Set[*model.Commodity]
	weights     map[time.Time]map[*model.Commodity]float64
}

func NewReport(reg *model.Registry, ds date.Partition) *Report {
	endDates := set.New[time.Time]()
	for _, d := range ds.EndDates() {
		endDates.Add(d)
	}
	return &Report{
		Registry:    reg,
		part:        ds,
		dates:       endDates,
		commodities: set.New[*model.Commodity](),
		weights:     make(map[time.Time]map[*commodity.Commodity]float64),
	}
}

func (r *Report) Add(d *journal.Day) error {
	if !r.dates.Has(d.Date) {
		return nil
	}
	var total float64
	for _, v := range d.Performance.V1 {
		total += v
	}
	weights := make(map[*model.Commodity]float64)
	for com, v := range d.Performance.V1 {
		weights[com] = v / total
		r.commodities.Add(com)
	}
	r.weights[d.Date] = weights
	return nil
}

type Renderer struct{}

func (rn *Renderer) Render(rep *Report) *table.Table {
	tbl := table.New(1, rep.part.Size())
	tbl.AddSeparatorRow()
	header := tbl.AddRow().AddText("Commodity", table.Center)
	for _, d := range rep.part.EndDates() {
		header.AddText(d.Format("2006-01-02"), table.Center)
	}
	tbl.AddSeparatorRow()

	for _, com := range rep.commodities.Sorted(commodity.Compare) {
		row := tbl.AddRow()
		row.AddText(com.Name(), table.Left)
		for _, d := range rep.part.EndDates() {
			n := rep.weights[d][com]
			if n != 0 {
				row.AddPercent(n)
			} else {
				row.AddEmpty()
			}
		}
	}
	tbl.AddSeparatorRow()

	return tbl
}
