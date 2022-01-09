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
func (calc Calculator) Perf(ctx context.Context, inCh <-chan *val.Day) (<-chan *DailyPerfValues, <-chan error) {
	resCh := make(chan *DailyPerfValues)
	errCh := make(chan error)

	go func() {
		defer close(resCh)
		defer close(errCh)

		var prev pcv

		for {
			d, ok, err := cpr.Pop(ctx, inCh)
			if !ok {
				break
			}
			if err != nil {
				return
			}

			dpr := calc.computeFlows(d)
			dpr.V1 = calc.valueByCommodity(d)
			dpr.V0 = prev
			prev = dpr.V1

			if cpr.Push(ctx, resCh, dpr) != nil {
				return
			}
		}
	}()
	return resCh, errCh
}

func (calc *Calculator) valueByCommodity(d *val.Day) pcv {
	var res = make(pcv)
	for pos, val := range d.Values {
		var t = pos.Account.Type()
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

func (calc *Calculator) computeFlows(step *val.Day) *DailyPerfValues {
	var internalInflows, internalOutflows, inflows, outflows pcv
	for _, trx := range step.Transactions {
		var cs = trx.Commodities()
		var gains pcv

		for _, pst := range trx.Postings {
			var value, _ = pst.Value.Float64()
			if calc.isPortfolioAccount(pst.Debit) {
				// TODO: handle marker booking for dividends (or more general?).
				switch pst.Credit.Type() {
				case journal.INCOME, journal.EXPENSES:
					if pst.TargetCommodity == nil {
						if len(cs) == 1 {
							// treat like a regular inflow
							get(&inflows)[pst.Commodity] += value
						} else {
							// create an unbalanced inflow
							get(&gains)[pst.Commodity] += value
						}
					} else if pst.Commodity != pst.TargetCommodity {
						// did not change the value of target, so we need to create
						// a virtual outflow
						get(&internalOutflows)[pst.TargetCommodity] -= value
						get(&internalInflows)[pst.Commodity] += value
					}
				case journal.ASSETS, journal.LIABILITIES:
					if !calc.Filter.MatchAccount(pst.Credit) {
						get(&inflows)[pst.Commodity] += value
					}
				case journal.EQUITY:
					if !pst.Amount.IsZero() && len(cs) > 1 {
						get(&gains)[pst.Commodity] += value
					}
				}
			}
			if calc.isPortfolioAccount(pst.Credit) {
				switch pst.Debit.Type() {
				case journal.INCOME, journal.EXPENSES:
					if pst.TargetCommodity == nil {
						if len(cs) == 1 {
							// treat like a regular inflow
							get(&outflows)[pst.Commodity] -= value
						} else {
							get(&gains)[pst.Commodity] -= value
						}
					} else if pst.Commodity != pst.TargetCommodity {
						// did not change the value of target, so we need to create
						// a virtual inflow
						get(&internalOutflows)[pst.Commodity] -= value
						get(&internalInflows)[pst.TargetCommodity] += value
					}
				case journal.ASSETS, journal.LIABILITIES:
					if !calc.Filter.MatchAccount(pst.Debit) {
						get(&outflows)[pst.Commodity] -= value
					}
				case journal.EQUITY:
					if !pst.Amount.IsZero() && len(cs) > 1 {
						get(&gains)[pst.Commodity] -= value
					}
				}
			}
		}
		if len(cs) > 1 {
			var diff float64
			for _, gain := range gains {
				diff += gain
			}
			var s = calc.determineStructure(gains)
			for c := range s {
				gains[c] -= diff / float64(len(s))
			}
			for c, gain := range gains {
				if gain > 0 {
					get(&internalInflows)[c] += gain
				} else if gain < 0 {
					get(&internalOutflows)[c] += gain
				}
			}
		}
	}
	return &DailyPerfValues{
		InternalInflow:  internalInflows,
		InternalOutflow: internalOutflows,
		Inflow:          inflows,
		Outflow:         outflows,
	}
}

func get(m *pcv) pcv {
	if *m == nil {
		*m = make(pcv)
	}
	return *m
}

func (calc Calculator) determineStructure(g pcv) map[*journal.Commodity]bool {
	var res = make(map[*journal.Commodity]bool)
	for c := range g {
		if !c.IsCurrency {
			res[c] = true
		}
	}
	if len(res) > 0 {
		return res
	}
	for c := range g {
		if c != calc.Valuation {
			res[c] = true
		}
	}
	if len(res) > 0 {
		return res
	}
	for c := range g {
		res[c] = true
	}
	return res
}

func (calc Calculator) isPortfolioAccount(a *journal.Account) bool {
	return (a.Type() == journal.ASSETS || a.Type() == journal.LIABILITIES) && calc.Filter.MatchAccount(a)
}

// perf = ( V1 - Outflow ) / ( V0 + Inflow )

// DailyPerfValues represents monetary values and flows in a period.
type DailyPerfValues struct {
	V0, V1, Inflow, Outflow, InternalInflow, InternalOutflow pcv
	Err                                                      error
}

func (dpv DailyPerfValues) performance() float64 {
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
