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
func (calc Calculator) Perf(l ledger.Ledger) <-chan DailyPerfValues {
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

// Valuator computes a daily value per commodity.
type Valuator struct {
	Balance *balance.Balance
	Filter  ledger.Filter
	Result  *DailyPerfValues
}

var _ ledger.Processor = (*Valuator)(nil)

// Process implements ledger.Processor.
func (v *Valuator) Process(_ *ledger.Day) error {
	var res = make(map[*ledger.Commodity]float64)
	for ca, val := range v.Balance.Values {
		var t = ca.Account.Type()
		if t != ledger.ASSETS && t != ledger.LIABILITIES {
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
	Filter    ledger.Filter
	Result    *DailyPerfValues
	Valuation *ledger.Commodity
}

var _ ledger.Processor = (*FlowComputer)(nil)

// Process implements ledger.Processor.
func (calc *FlowComputer) Process(step *ledger.Day) error {
	var (
		internalInflows  = make(map[*ledger.Commodity]float64)
		internalOutflows = make(map[*ledger.Commodity]float64)
		inflows          = make(map[*ledger.Commodity]float64)
		outflows         = make(map[*ledger.Commodity]float64)
	)
	for _, trx := range step.Transactions {
		var gains = make(map[*ledger.Commodity]float64)
		for _, pst := range trx.Postings {
			var value, _ = pst.Value.Float64()
			if calc.isPortfolioAccount(pst.Debit) {
				// TODO: handle marker booking for dividends (or more general?).
				switch pst.Credit.Type() {
				case ledger.INCOME, ledger.EXPENSES:
					if pst.Commodity != calc.Valuation {
						internalInflows[pst.Commodity] += value
						internalOutflows[calc.Valuation] -= value
					}
				case ledger.ASSETS, ledger.LIABILITIES:
					if !calc.Filter.MatchAccount(pst.Credit) {
						inflows[pst.Commodity] += value
					}
				case ledger.EQUITY:
					if !pst.Amount.IsZero() {
						gains[pst.Commodity] += value
					}
				}
			}
			if calc.isPortfolioAccount(pst.Credit) {
				switch pst.Debit.Type() {
				case ledger.INCOME, ledger.EXPENSES:
					if pst.Commodity != calc.Valuation {
						internalOutflows[pst.Commodity] -= value
						internalInflows[calc.Valuation] += value
					}
				case ledger.ASSETS, ledger.LIABILITIES:
					if !calc.Filter.MatchAccount(pst.Debit) {
						outflows[pst.Commodity] -= value
					}
				case ledger.EQUITY:
					if !pst.Amount.IsZero() {
						gains[pst.Commodity] -= value
					}
				}
			}
		}
		var totalGains, totalLosses float64
		for _, gain := range gains {
			if gain > 0 {
				totalGains += gain
			} else {
				totalLosses += gain
			}
		}
		var diff = totalGains + totalLosses
		for c, gain := range gains {
			if gain > 0 {
				internalInflows[c] += gain * (1 - 0.5*diff/totalGains)
			} else if gain < 0 {
				internalOutflows[c] += gain * (1 - 0.5*diff/totalLosses)
			}
		}
	}
	calc.Result.InternalInflow = internalInflows
	calc.Result.InternalOutflow = internalOutflows
	calc.Result.Inflow = inflows
	calc.Result.Outflow = outflows

	return nil
}

func (calc FlowComputer) isPortfolioAccount(a *ledger.Account) bool {
	return (a.Type() == ledger.ASSETS || a.Type() == ledger.LIABILITIES) && calc.Filter.MatchAccount(a)
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
