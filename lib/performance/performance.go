package performance

import (
	"fmt"

	"github.com/sboehler/knut/lib/balance"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
)

// Calculator calculates portfolio performance
type Calculator struct {
	Valuation *journal.Commodity
	Filter    journal.Filter
}

// Perf computes portfolio performance.
func (calc Calculator) Perf(l *ast.PAST) <-chan DailyPerfValues {
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

	// TODO: make this a ast.Process step!
	return make(chan DailyPerfValues)
}

// Valuator computes a daily value per commodity.
type Valuator struct {
	Balance *balance.Balance
	Filter  journal.Filter
	Result  *DailyPerfValues
}

var _ ast.Processor = (*Valuator)(nil)

// Process implements ast.Processor.
func (v *Valuator) Process(_ *ast.Day) error {
	var res = make(pcv)
	for ca, val := range v.Balance.Values {
		var t = ca.Account.Type()
		if t != journal.ASSETS && t != journal.LIABILITIES {
			continue
		}
		if !v.Filter.MatchAccount(ca.Account) || !v.Filter.MatchCommodity(ca.Commodity) {
			continue
		}
		valF, _ := val.Float64()
		res[ca.Commodity] = res[ca.Commodity] + valF
	}
	v.Result.V0 = v.Result.V1
	v.Result.V1 = res
	return nil
}

// FlowComputer computes internal and external value flows for a portfolio.
type FlowComputer struct {
	Filter    journal.Filter
	Result    *DailyPerfValues
	Valuation *journal.Commodity
}

var _ ast.Processor = (*FlowComputer)(nil)

// pcv is a per-commodity value.
type pcv map[*journal.Commodity]float64

// Process implements ast.Processor.
func (calc *FlowComputer) Process(step *ast.Day) error {
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
	calc.Result.InternalInflow = internalInflows
	calc.Result.InternalOutflow = internalOutflows
	calc.Result.Inflow = inflows
	calc.Result.Outflow = outflows

	return nil
}

func get(m *pcv) pcv {
	if *m == nil {
		*m = make(pcv)
	}
	return *m
}

func (calc FlowComputer) determineStructure(g pcv) map[*journal.Commodity]bool {
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

func (calc FlowComputer) isPortfolioAccount(a *journal.Account) bool {
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
