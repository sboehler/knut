package performance

import (
	"fmt"
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

// Process computes portfolio performance.
func (calc Calculator) Process(d *journal.Day, next func(*journal.Day)) error {
	var prev pcv
	dpr := calc.computeFlows(d)
	dpr.V0 = prev
	dpr.V1 = calc.valueByCommodity(d)
	prev = dpr.V1
	d.Performance = dpr
	next(d)
	return nil
}

// Sink implements Sink.
func (calc Calculator) Sink(d *journal.Day) {
	fmt.Printf("%v: %.1f%%\n", d.Date.Format("2006-01-02"), 100*(Performance(d.Performance)-1))
}

func (calc *Calculator) valueByCommodity(d *journal.Day) pcv {
	res := make(pcv)
	for pos, val := range d.Value {
		if !pos.Account.IsAL() {
			continue
		}
		if !calc.AccountFilter(pos.Account) || !calc.CommodityFilter(pos.Commodity) {
			continue
		}
		valF, _ := val.Float64()
		res[pos.Commodity] = res[pos.Commodity] + valF
	}
	return res
}

// pcv is a per-commodity value.
type pcv map[*journal.Commodity]float64

func (calc *Calculator) computeFlows(day *journal.Day) *journal.Performance {

	var (
		internalInflows, internalOutflows, inflows, outflows pcv
		portfolioFlows                                       float64
	)

	for _, trx := range day.Transactions {

		// We make the convention that flows per transaction and commodity are
		// either positive or negative, but not both.
		var flows, internalFlows pcv

		for _, pst := range trx.Postings {
			value, _ := pst.Amount.Float64()
			var otherAccount *journal.Account

			if calc.isPortfolioAccount(pst.Credit) && !calc.isPortfolioAccount(pst.Debit) {
				// portfolio outflow
				otherAccount = pst.Debit
				value = -value
			} else if !calc.isPortfolioAccount(pst.Credit) && calc.isPortfolioAccount(pst.Debit) {
				// portfolio inflow
				otherAccount = pst.Credit
			} else {
				// nothing to do, as the posting does not affect the portfolio
				continue
			}

			// tgts contains the commodities among which the performance effects of this
			// transaction should be split: non-currencies > currencies > valuation currency.
			tgts := calc.pickTargets(pst.Targets)

			if otherAccount.IsAL() || tgts == nil {
				// no effect: regular flow into or out of the portfolio
				get(&flows)[pst.Commodity] += value
				continue
			}
			if len(tgts) == 1 && tgts[0] == pst.Commodity {
				// performance effect on native commodity
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
	return &journal.Performance{
		InternalInflow:   internalInflows,
		InternalOutflow:  internalOutflows,
		Inflow:           inflows,
		Outflow:          outflows,
		PortfolioInflow:  math.Max(0, portfolioFlows),
		PortfolioOutflow: math.Min(0, portfolioFlows),
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

func (calc Calculator) pickTargets(tgts []*journal.Commodity) []*journal.Commodity {
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
		if c != calc.Valuation {
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
