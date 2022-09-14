package performance

import (
	"context"
	"fmt"
	"math"

	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/common/filter"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
)

// Calculator calculates portfolio performance
type Calculator struct {
	Context         journal.Context
	Valuation       *journal.Commodity
	AccountFilter   filter.Filter[*journal.Account]
	CommodityFilter filter.Filter[*journal.Commodity]
}

// Process computes portfolio performance.
func (calc Calculator) Process(ctx context.Context, inCh <-chan *ast.Day, outCh chan<- *ast.Day) error {
	var prev pcv
	return cpr.Consume(ctx, inCh, func(d *ast.Day) error {
		dpr := calc.computeFlows(d)
		dpr.V0 = prev
		dpr.V1 = calc.valueByCommodity(d)
		prev = dpr.V1
		d.Performance = dpr

		return cpr.Push(ctx, outCh, d)
	})
}

// Sink implements Sink.
func (calc Calculator) Sink(ctx context.Context, inCh <-chan *ast.Day) error {
	return cpr.Consume(ctx, inCh, func(p *ast.Day) error {
		fmt.Printf("%v: %.1f%%\n", p.Date.Format("2006-01-02"), 100*(Performance(p.Performance)-1))
		return nil
	})
}

func (calc *Calculator) valueByCommodity(d *ast.Day) pcv {
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

func (calc *Calculator) computeFlows(day *ast.Day) *ast.Performance {

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
			tgts := calc.determineStructure(pst.Targets)

			switch otherAccount.Type() {

			case journal.INCOME, journal.EXPENSES, journal.EQUITY:
				if tgts == nil {
					// no effect: regular flow into or out of the portfolio
					get(&flows)[pst.Commodity] += value
				} else if len(tgts) == 0 {
					// performance effect on portfolio, not allocated to a specific commodity
					get(&internalFlows)[pst.Commodity] += value
					portfolioFlows -= value
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
	return &ast.Performance{
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

func (calc Calculator) determineStructure(tgts []*journal.Commodity) []*journal.Commodity {
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
func Performance(dpv *ast.Performance) float64 {
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
