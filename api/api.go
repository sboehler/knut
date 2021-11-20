// Package api provides the knut web API.
package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/sboehler/knut/lib/balance"
	"github.com/sboehler/knut/lib/date"
	"github.com/sboehler/knut/lib/ledger"
	"github.com/sboehler/knut/lib/model/accounts"
	"github.com/sboehler/knut/lib/model/commodities"
	"github.com/sboehler/knut/lib/parser"
	"github.com/shopspring/decimal"
)

// New instantiates the API handler.
func New(file string) http.Handler {
	var s = http.NewServeMux()
	s.Handle("/balance", handler{file})
	return s
}

// handler handles HTTP.
type handler struct {
	File string
}

func (s handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var (
		ppl *pipeline
		err error
	)
	if ppl, err = buildPipeline(s.File, r.URL.Query()); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err = ppl.process(w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type pipeline struct {
	Accounts       *accounts.Accounts
	Parser         parser.RecursiveParser
	Filter         ledger.Filter
	BalanceBuilder balance.Builder
}

func buildPipeline(file string, query url.Values) (*pipeline, error) {
	var (
		period                            *date.Period
		commoditiesFilter, accountsFilter *regexp.Regexp
		from, to                          *time.Time
		last                              int
		valuation                         *commodities.Commodity
		diff, close                       bool
		err                               error
	)
	if period, err = parsePeriod(query, "period"); err != nil {
		return nil, err
	}
	if commoditiesFilter, err = parseRegex(query, "commodity"); err != nil {
		return nil, err
	}
	if accountsFilter, err = parseRegex(query, "account"); err != nil {
		return nil, err
	}
	if from, err = parseDate(query, "from"); err != nil {
		return nil, err
	}
	if to, err = parseDate(query, "to"); err != nil {
		return nil, err
	}
	if last, err = parseInt(query, "last"); err != nil {
		return nil, err
	}
	if valuation, err = parseCommodity(query, "valuation"); err != nil {
		return nil, err
	}
	if diff, err = parseBool(query, "diff"); err != nil {
		return nil, err
	}
	if close, err = parseBool(query, "close"); err != nil {
		return nil, err
	}

	return &pipeline{
		Parser: parser.RecursiveParser{
			Accounts: accounts.New(),
			File:     file,
		},
		Filter: ledger.Filter{
			AccountsFilter:    accountsFilter,
			CommoditiesFilter: commoditiesFilter,
		},
		BalanceBuilder: balance.Builder{
			From:      from,
			To:        to,
			Period:    period,
			Last:      last,
			Valuation: valuation,
			Close:     close,
			Diff:      diff,
		},
	}, nil
}

func (ppl *pipeline) process(w io.Writer) error {
	l, err := ppl.Parser.BuildLedger(ppl.Filter)
	if err != nil {
		return err
	}
	b, err := ppl.BalanceBuilder.Build(l)
	if err != nil {
		return err
	}
	var (
		j = balanceToJSON(b)
		e = json.NewEncoder(w)
	)
	return e.Encode(j)
}

var periods = map[string]date.Period{
	"days":     date.Daily,
	"weeks":    date.Weekly,
	"months":   date.Monthly,
	"quarters": date.Quarterly,
	"years":    date.Yearly,
}

func parsePeriod(query url.Values, key string) (*date.Period, error) {
	var (
		period date.Period
		value  string
		ok     bool
		err    error
	)
	if value, ok, err = getOne(query, key); err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	if period, ok = periods[value]; !ok {
		return nil, fmt.Errorf("invalid period %q", value)
	}
	return &period, nil
}

func parseRegex(query url.Values, key string) (*regexp.Regexp, error) {
	var (
		s      string
		ok     bool
		err    error
		result *regexp.Regexp
	)
	if s, ok, err = getOne(query, key); err != nil {
		return nil, err
	}
	if ok {
		if result, err = regexp.Compile(s); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func parseDate(query url.Values, key string) (*time.Time, error) {
	var (
		s      string
		ok     bool
		err    error
		result *time.Time
	)
	if s, ok, err = getOne(query, key); err != nil {
		return nil, err
	}
	if ok {
		var t time.Time
		if t, err = time.Parse("2006-01-02", s); err != nil {
			return nil, err
		}
		result = &t
	}
	return result, nil
}

func parseInt(query url.Values, key string) (int, error) {
	var (
		s      string
		ok     bool
		err    error
		result int
	)
	if s, ok, err = getOne(query, key); err != nil {
		return result, err
	}
	if !ok {
		return 0, nil
	}
	return strconv.Atoi(s)
}

func parseBool(query url.Values, key string) (bool, error) {
	var (
		s      string
		ok     bool
		err    error
		result bool
	)
	if s, ok, err = getOne(query, key); err != nil {
		return result, err
	}
	if !ok {
		return result, nil
	}
	return strconv.ParseBool(s)
}

func parseCommodity(query url.Values, key string) (*commodities.Commodity, error) {
	var (
		s   string
		ok  bool
		err error
	)
	if s, ok, err = getOne(query, key); err != nil || !ok {
		return nil, err
	}
	return commodities.Get(s)
}

func getOne(query url.Values, key string) (string, bool, error) {
	values, ok := query[key]
	if !ok {
		return "", ok, nil
	}
	if len(values) != 1 {
		return "", false, fmt.Errorf("expected one value for query parameter %q, got %v", key, values)
	}
	return values[0], true, nil
}

type jsonBalance struct {
	Valuation       *commodities.Commodity
	Dates           []time.Time
	Amounts, Values map[string]map[string][]decimal.Decimal
}

func balanceToJSON(bs []*balance.Balance) *jsonBalance {
	var res = jsonBalance{
		Valuation: bs[0].Valuation,
		Amounts:   make(map[string]map[string][]decimal.Decimal),
		Values:    make(map[string]map[string][]decimal.Decimal),
	}
	var wg sync.WaitGroup
	for i, b := range bs {
		res.Dates = append(res.Dates, b.Date)
		wg.Add(2)
		i := i
		b := b
		go func() {
			defer wg.Done()
			for pos, amount := range b.Amounts {
				insert(res.Amounts, i, len(bs), pos, amount)
			}
		}()
		go func() {
			defer wg.Done()
			for pos, value := range b.Amounts {
				insert(res.Values, i, len(bs), pos, value)
			}
		}()
		wg.Wait()
	}
	return &res
}

func insert(m map[string]map[string][]decimal.Decimal, i int, n int, pos balance.CommodityAccount, amount decimal.Decimal) {
	a, ok := m[pos.Account.String()]
	if !ok {
		a = make(map[string][]decimal.Decimal)
		m[pos.Account.String()] = a
	}
	c, ok := a[pos.Commodity.String()]
	if !ok {
		c = make([]decimal.Decimal, n)
		a[pos.Commodity.String()] = c
	}
	c[i] = amount
}
