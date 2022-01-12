package performance

import (
	"context"
	"fmt"

	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/val"
)

// Calculator calculates portfolio performance
type Calculator struct {
	Context   journal.Context
	Valuation *journal.Commodity
	Filter    journal.Filter
}

// Perf computes portfolio performance.
func (calc Calculator) Perf(ctx context.Context, inCh <-chan *val.Day) (<-chan *PerfPeriod, <-chan error) {
	resCh := make(chan *PerfPeriod)
	errCh := make(chan error)

	go func() {
		defer close(resCh)
		defer close(errCh)

		var prev pcv

		for {
			d, ok, err := cpr.Pop(ctx, inCh)
			if !ok || err != nil {
				return
			}

			dpr := calc.computeFlows(d)
			dpr.V0 = prev
			dpr.V1 = calc.valueByCommodity(d)
			prev = dpr.V1

			if cpr.Push(ctx, resCh, dpr) != nil {
				return
			}
		}
	}()
	return resCh, errCh
}

func (calc *Calculator) valueByCommodity(d *val.Day) pcv {
	res := make(pcv)
	for pos, val := range d.Values {
		t := pos.Account.Type()
		if t != journal.ASSETS && t != journal.LIABILITIES {
			continue
		}
		if !calc.Filter.MatchAccount(pos.Account) || !calc.Filter.MatchCommodity(pos.Commodity) {
			continue
		}
		valF, _ := val.Float64()
		res[pos.Commodity] = res[pos.Commodity] + valF
	}
	return res
}

// pcv is a per-commodity value.
type pcv map[*journal.Commodity]float64

func (calc *Calculator) computeFlows(step *val.Day) *PerfPeriod {

	var internalInflows, internalOutflows, inflows, outflows pcv

	for _, trx := range step.Transactions {

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
			tgts := calc.determineStructure(pst.Targets)

			switch otherAccount.Type() {

			case journal.INCOME, journal.EXPENSES, journal.EQUITY:
				if len(tgts) == 0 {
					// no effect: regular flow into or out of the portfolio
					get(&flows)[pst.Commodity] += value
				} else if len(tgts) > 1 || len(tgts) == 1 && tgts[0] != pst.Commodity {
					// effect on multiple commodities: re-allocate the flows among the target commodities
					l := float64(len(tgts))
					intf := get(&internalFlows)
					for _, com := range tgts {
						intf[com] -= value / l
					}
					intf[pst.Commodity] += value
				}

			case journal.ASSETS, journal.LIABILITIES:
				get(&flows)[pst.Commodity] += value
			}
		}

		split(flows, &inflows, &outflows)
		split(internalFlows, &internalInflows, &internalOutflows)
	}
	return &PerfPeriod{
		InternalInflow:  internalInflows,
		InternalOutflow: internalOutflows,
		Inflow:          inflows,
		Outflow:         outflows,
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

func (calc Calculator) determineStructure(g []*journal.Commodity) []*journal.Commodity {
	var res []*journal.Commodity
	for _, c := range g {
		if !c.IsCurrency {
			res = append(res, c)
		}
	}
	if len(res) > 0 {
		return res
	}
	for _, c := range g {
		if c != calc.Valuation {
			res = append(res, c)
		}
	}
	if len(res) > 0 {
		return res
	}
	for _, c := range g {
		res = append(res, c)
	}
	return res
}

func (calc Calculator) isPortfolioAccount(a *journal.Account) bool {
	return (a.Type() == journal.ASSETS || a.Type() == journal.LIABILITIES) && calc.Filter.MatchAccount(a)
}

// perf = ( V1 - Outflow ) / ( V0 + Inflow )

// PerfPeriod represents monetary values and flows in a period.
type PerfPeriod struct {
	V0, V1, Inflow, Outflow, InternalInflow, InternalOutflow pcv
	Err                                                      error
}

func (dpv PerfPeriod) performance() float64 {
	var v0, v1, inflow, outflow, internalInflow, internalOutflow float64
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
	for _, v := range dpv.InternalInflow {
		internalInflow += v
	}
	for _, v := range dpv.InternalOutflow {
		internalOutflow += v
	}
	fmt.Printf("%.0f %.0f %.0f %.0f %.0f %.0f\n", v0, v1, inflow, outflow, internalInflow, internalOutflow)
	return (v1 - outflow) / (v0 + inflow)
}
