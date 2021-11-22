package performance

import (
	"fmt"

	"github.com/sboehler/knut/lib/balance"
	"github.com/sboehler/knut/lib/ledger"
)

// Calculator calculates portfolio performance
type Calculator struct {
	Valuation *ledger.Commodity
	Filter    ledger.Filter
}

// Perf computes portfolio performance.
func (b Calculator) Perf(l ledger.Ledger) <-chan DailyPerfValues {
	// var (
	// 	bal               = balance.New(l.Context, b.Valuation)
	// 	ps                = make(prices.Prices)
	// 	np                prices.NormalizedPrices
	// 	ch                = make(chan DailyPerfValues)
	// 	previous, current DailyPerfValues
	// )
	// go func() {
	// 	defer close(ch)
	// 	for _, step := range l.Days {
	// 		for _, p := range step.Prices {
	// 			ps.Insert(p)
	// 		}
	// 		np = ps.Normalize(b.Valuation)
	// 		if current.Err = bal.Update(step, np, false); current.Err != nil {
	// 			ch <- current
	// 			return
	// 		}
	// 		current = b.computeFlows(step)
	// 		current.V0 = previous.V1
	// 		current.V1 = b.computeValue(bal)
	// 		ch <- current
	// 		fmt.Printf("%s %.4f\n", step.Date, current.performance())
	// 		previous = current
	// 	}
	// }()
	// return ch

	// TODO: make this a ledger.Process step!
	return make(chan DailyPerfValues)
}

func (b Calculator) computeValue(bal *balance.Balance) map[*ledger.Commodity]float64 {
	var res = make(map[*ledger.Commodity]float64)
	for ca, val := range bal.Values {
		if !b.isPortfolioAccount(ca.Account) {
			continue
		}
		if !b.Filter.MatchCommodity(ca.Commodity) {
			continue
		}
		valF, _ := val.Float64()
		res[ca.Commodity] = res[ca.Commodity] + valF
	}
	return res
}

func (b Calculator) computeFlows(step *ledger.Day) DailyPerfValues {
	var flow = DailyPerfValues{
		Inflow:          make(map[*ledger.Commodity]float64),
		Outflow:         make(map[*ledger.Commodity]float64),
		InternalInflow:  make(map[*ledger.Commodity]float64),
		InternalOutflow: make(map[*ledger.Commodity]float64),
	}
	for _, trx := range step.Transactions {
		var include bool
		for _, pst := range trx.Postings {
			if b.isPortfolioAccount(pst.Credit) || b.isPortfolioAccount(pst.Debit) {
				include = true
				break
			}
		}
		if !include {
			continue
		}
		var internal = make(map[*ledger.Commodity]float64)
		for _, pst := range trx.Postings {
			var valF, _ = pst.Value.Float64()
			switch pst.Credit.Type() {
			case ledger.INCOME, ledger.EXPENSES:
				if pst.Commodity != b.Valuation {
					flow.InternalInflow[pst.Commodity] += valF
					flow.InternalOutflow[b.Valuation] -= valF
				}
			case ledger.ASSETS, ledger.LIABILITIES:
				if !b.Filter.MatchAccount(pst.Credit) {
					flow.Inflow[pst.Commodity] += valF
				}
			case ledger.EQUITY:
				if !pst.Amount.IsZero() {
					internal[pst.Commodity] += valF
				}
			}
			switch pst.Debit.Type() {
			case ledger.INCOME, ledger.EXPENSES:
				if pst.Commodity != b.Valuation {
					flow.InternalOutflow[pst.Commodity] -= valF
					flow.InternalInflow[b.Valuation] += valF
				}
			case ledger.ASSETS, ledger.LIABILITIES:
				if !b.Filter.MatchAccount(pst.Debit) {
					flow.Outflow[pst.Commodity] -= valF
				}
			case ledger.EQUITY:
				if !pst.Amount.IsZero() {
					internal[pst.Commodity] -= valF
				}
			}
		}
		if len(internal) > 0 {
			fmt.Println(internal)
		}
		var totalGains, totalLosses float64
		for _, gain := range internal {
			if gain > 0 {
				totalGains += gain
			} else {
				totalLosses += gain
			}
		}
		var diff = totalGains + totalLosses
		if len(internal) > 0 {
			fmt.Println(totalGains, totalLosses, flow.InternalInflow, flow.InternalOutflow, diff)
		}
		for c, gain := range internal {
			if gain > 0 {
				flow.InternalInflow[c] += gain * (1 - 0.5*diff/totalGains)
			} else if gain < 0 {
				flow.InternalOutflow[c] += gain * (1 - 0.5*diff/totalLosses)
			}
		}
		if len(internal) > 0 {
			fmt.Println(flow.InternalInflow, flow.InternalOutflow)
		}
	}
	return flow
}

func (b Calculator) isPortfolioAccount(a *ledger.Account) bool {
	return (a.Type() == ledger.ASSETS || a.Type() == ledger.LIABILITIES) && b.Filter.MatchAccount(a)
}

// perf = ( V1 - Outflow ) / ( V0 + Inflow )

// DailyPerfValues represents monetary values and flows in a period.
type DailyPerfValues struct {
	V0, V1, Inflow, Outflow, InternalInflow, InternalOutflow map[*ledger.Commodity]float64
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
