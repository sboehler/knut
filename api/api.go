// Package api provides the knut web API.
package api

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/sboehler/knut/lib/balance"
	"github.com/sboehler/knut/lib/date"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/ledger"
	"github.com/sboehler/knut/lib/model/commodities"
	"github.com/sboehler/knut/lib/report"
	"github.com/sboehler/knut/lib/table"
)

// Handler handles HTTP.
type Handler struct {
	File string
}

func (s Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var (
		ppl *pipeline
		err error
	)
	if ppl, err = buildPipeline(s.File, r.URL.Query()); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err = ppl.process(w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type pipeline struct {
	Journal        journal.Journal
	LedgerFilter   ledger.Filter
	BalanceBuilder balance.Builder
	ReportBuilder  report.Builder
	ReportRenderer report.Renderer
	TextRenderer   table.TextRenderer
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

	var (
		journal = journal.Journal{
			File: file,
		}
		ledgerFilter = ledger.Filter{
			CommoditiesFilter: commoditiesFilter,
			AccountsFilter:    accountsFilter,
		}

		balanceBuilder = balance.Builder{
			From:      from,
			To:        to,
			Period:    period,
			Last:      last,
			Valuation: valuation,
			Close:     close,
			Diff:      diff,
		}

		reportBuilder = report.Builder{
			// Value:    valuation != nil,
			// Collapse: collapse,
		}

		reportRenderer = report.Renderer{
			Commodities: true, // showCommodities || valuation == nil,
		}

		tableRenderer = table.TextRenderer{
			// Color:     color,
			// Thousands: thousands,
			// Round:     digits,
		}
	)

	return &pipeline{
		Journal:        journal,
		LedgerFilter:   ledgerFilter,
		BalanceBuilder: balanceBuilder,
		ReportBuilder:  reportBuilder,
		ReportRenderer: reportRenderer,
		TextRenderer:   tableRenderer,
	}, nil
}

func (ppl *pipeline) process(w io.Writer) error {
	l, err := ledger.FromDirectives(ppl.LedgerFilter, ppl.Journal.Parse())
	if err != nil {
		return err
	}
	b, err := ppl.BalanceBuilder.Build(l)
	if err != nil {
		return err
	}
	r, err := ppl.ReportBuilder.Build(b)
	if err != nil {
		return err
	}
	return ppl.TextRenderer.Render(ppl.ReportRenderer.Render(r), w)
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
		s      string
		ok     bool
		err    error
		result *commodities.Commodity
	)
	if s, ok, err = getOne(query, key); err != nil {
		return nil, err
	}
	if ok {
		result = commodities.Get(s)
	}
	return result, nil
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
