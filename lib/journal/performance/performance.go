package performance

import (
	"math"

	"github.com/sboehler/knut/lib/common/filter"
	"github.com/sboehler/knut/lib/journal"
)

// Calculator calculates portfolio performance
type Calculator struct {
	Context         journal.Context
	Valuation       *journal.Commodity
	AccountFilter   filter.Filter[*journal.Account]
	CommodityFilter filter.Filter[*journal.Commodity]
}

// ComputeValues computes portfolio performance.
func (calc *Calculator) ComputeValues() func(d *journal.Day) error {
	var prev pcv
	values := make(journal.Amounts)
	return func(d *journal.Day) error {
		if d.Performance == nil {
			d.Performance = new(journal.Performance)
		}
		d.Performance.V0 = prev

		for _, t := range d.Transactions {
			for _, p := range t.Postings {
				if !calc.CommodityFilter(p.Commodity) {
					continue
				}
				if !calc.isPortfolioAccount(p.Account) {
					continue
				}
				k := journal.CommodityKey(p.Commodity)
				values.Add(k, p.Value)
				if values[k].IsZero() {
					delete(values, k)
				}
			}
		}
		if len(values) == 0 {
			return nil
		}
		prev = nil
		for k, v := range values {
			f, _ := v.Float64()
			get(&prev)[k.Commodity] += f
		}
		d.Performance.V1 = prev
		return nil
	}
}

// pcv is a per-commodity value.
type pcv map[*journal.Commodity]float64

func (calc *Calculator) ComputeFlows() journal.DayFn {
	return func(day *journal.Day) error {
		var (
			internalInflows, internalOutflows, inflows, outflows pcv
			portfolioFlows                                       float64
		)
		for _, trx := range day.Transactions {

			// We make the convention that flows per transaction and commodity are
			// either positive or negative, but not both.
			var flows, internalFlows pcv

			for _, pst := range trx.Postings {

				if !calc.isPortfolioAccount(pst.Account) {
					// not a portfolio booking - no performance impact.
					continue
				}

				if calc.isPortfolioAccount(pst.Other) {
					// transfer between portfolio accounts - no performance impact.
					continue
				}
				// tgts contains the commodities among which the performance effects of this
				// transaction should be split: non-currencies > currencies > valuation currency.
				tgts := pickTargets(calc.Valuation, pst.Targets)

				if len(tgts) == 1 && tgts[0] == pst.Commodity {
					// performance effect on native commodity
					continue
				}

				value, _ := pst.Amount.Float64()
				if tgts == nil {
					// regular flow into or out of the portfolio
					get(&flows)[pst.Commodity] += value
					continue
				}
				intf := get(&internalFlows)
				intf[pst.Commodity] += value
				if len(tgts) == 0 {
					// performance effect on portfolio, not allocated to a specific commodity
					portfolioFlows -= value
				} else {
					// effect on multiple commodities: re-allocate the flows among the target commodities
					l := float64(len(tgts))
					for _, com := range tgts {
						intf[com] -= value / l
					}
				}
			}

			split(flows, &inflows, &outflows)
			split(internalFlows, &internalInflows, &internalOutflows)
		}
		if day.Performance == nil {
			day.Performance = new(journal.Performance)
		}
		day.Performance.InternalInflow = internalInflows
		day.Performance.InternalOutflow = internalOutflows
		day.Performance.Inflow = inflows
		day.Performance.Outflow = outflows
		day.Performance.PortfolioInflow = math.Max(0, portfolioFlows)
		day.Performance.PortfolioOutflow = math.Min(0, portfolioFlows)
		return nil
	}
}

func split(flows pcv, in, out *pcv) {
	for c, f := range flows {
		if f > 0 {
			get(in)[c] += f
		} else if f < 0 {
			get(out)[c] += f
		}
	}
}

func get(m *pcv) pcv {
	if *m == nil {
		*m = make(pcv)
	}
	return *m
}

func pickTargets(valuation *journal.Commodity, tgts []*journal.Commodity) []*journal.Commodity {
	if len(tgts) == 0 {
		return tgts
	}
	var res []*journal.Commodity

	// collect non-currencies
	for _, c := range tgts {
		if !c.IsCurrency {
			res = append(res, c)
		}
	}
	if len(res) > 0 {
		return res
	}

	// collect currencies != valuation
	for _, c := range tgts {
		if c != valuation {
			res = append(res, c)
		}
	}
	if len(res) > 0 {
		return res
	}

	// return all
	return tgts
}

func (calc Calculator) isPortfolioAccount(a *journal.Account) bool {
	return a.IsAL() && calc.AccountFilter(a)
}

// perf = ( V1 - Outflow ) / ( V0 + Inflow )

// Performance computes the portfolio performance.
func Performance(dpv *journal.Performance) float64 {
	var (
		v0, v1          float64
		inflow, outflow = dpv.PortfolioInflow, dpv.PortfolioOutflow
	)
	for _, v := range dpv.V0 {
		v0 += v
	}
	for _, v := range dpv.V1 {
		v1 += v
	}
	for _, v := range dpv.Inflow {
		inflow += v
	}
	for _, v := range dpv.Outflow {
		outflow += v
	}
	if v0 == v1 && inflow == 0 && outflow == 0 {
		return 1
	}
	return (v1 - outflow) / (v0 + inflow)
}
