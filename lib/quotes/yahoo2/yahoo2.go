// Copyright 2021 Silvio Böhler
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package yahoo2

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"
)

const yahooURL string = "https://query2.finance.yahoo.com/v8/finance/chart"

// Quote represents a quote on a given day.
type Quote struct {
	Date     time.Time
	Open     float64
	High     float64
	Low      float64
	Close    float64
	AdjClose float64
	Volume   int
}

// Client is a client for Yahoo! quotes.
type Client struct {
	url string
}

// New creates a new client with the default URL.
func New() Client {
	return Client{yahooURL}
}

// Fetch fetches a set of quotes
func (c *Client) Fetch(sym string, t0, t1 time.Time) ([]Quote, error) {
	u, err := createURL(c.url, sym, t0, t1)
	if err != nil {
		return nil, fmt.Errorf("error creating URL for symbol %s: %w", sym, err)
	}
	resp, err := http.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("error fetching data from URL %s: %w", u.String(), err)
	}
	defer resp.Body.Close()
	quote, err := decodeResponse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error decoding response for symbol %s (url: %s): %w", sym, u, err)
	}
	return quote, nil
}

// createURL creates a URL for the given root URL and parameters.
func createURL(rootURL, sym string, t0, t1 time.Time) (*url.URL, error) {
	u, err := url.Parse(rootURL)
	if err != nil {
		return u, err
	}
	u.Path = path.Join(u.Path, url.PathEscape(sym))
	u.RawQuery = url.Values{
		"events":   {"history"},
		"interval": {"1d"},
		"period1":  {fmt.Sprint(t0.Unix())},
		"period2":  {fmt.Sprint(t1.Unix())},
	}.Encode()
	return u, nil
}

// decodeResponse takes a reader for the response and returns
// the parsed quotes.
func decodeResponse(r io.Reader) ([]Quote, error) {
	d := json.NewDecoder(r)
	var body jbody
	if err := d.Decode(&body); err != nil {
		return nil, err
	}
	result := body.Chart.Result[0]
	loc, err := time.LoadLocation(result.Meta.ExchangeTimezoneName)
	if err != nil {
		return nil, fmt.Errorf("unknown time zone: %s", result.Meta.ExchangeTimezoneName)
	}
	var res []Quote
	for i, ts := range body.Chart.Result[0].Timestamp {
		date := time.Unix(int64(ts), 0).In(loc)
		q := Quote{
			Date:     time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC),
			Open:     result.Indicators.Quote[0].Open[i],
			Close:    result.Indicators.Quote[0].Close[i],
			High:     result.Indicators.Quote[0].High[i],
			Low:      result.Indicators.Quote[0].Low[i],
			AdjClose: result.Indicators.Adjclose[0].Adjclose[i],
			Volume:   result.Indicators.Quote[0].Volume[i],
		}
		if q.Close > 0 {
			res = append(res, q)
		}
	}
	return res, nil
}

type jbody struct {
	Chart jchart `json:"chart"`
}
type jchart struct {
	Result []jresult `json:"result"`
}

type jresult struct {
	Meta       jmeta       `json:"meta"`
	Timestamp  []int       `json:"timestamp"`
	Indicators jindicators `json:"indicators"`
}

type jmeta struct {
	ExchangeTimezoneName string `json:"exchangeTimezoneName"`
}

type jindicators struct {
	Quote    []jquote    `json:"quote"`
	Adjclose []jadjclose `json:"adjclose"`
}

type jquote struct {
	Volume []int     `json:"volume"`
	High   []float64 `json:"high"`
	Close  []float64 `json:"close"`
	Low    []float64 `json:"low"`
	Open   []float64 `json:"open"`
}

type jadjclose struct {
	Adjclose []float64 `json:"adjclose"`
}
