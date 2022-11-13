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

package yahoo

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"
)

const yahooURL string = "https://query1.finance.yahoo.com/v7/finance/download"

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
		return nil, err
	}
	resp, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return decodeResponse(resp.Body)
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
func decodeResponse(r io.ReadCloser) ([]Quote, error) {
	csvReader := csv.NewReader(r)
	csvReader.FieldsPerRecord = 7
	// skip header
	if _, err := csvReader.Read(); err != nil {
		return nil, err
	}
	// read lines
	var res []Quote
	for {
		r, err := csvReader.Read()
		if err == io.EOF {
			return res, nil
		}
		if err != nil {
			return nil, err
		}
		quote, ok, err := recordToQuote(r)
		if err != nil {
			return nil, err
		}
		if ok {
			res = append(res, quote)
		}
	}
}

// recordToQuote decodes one line of the response CSV.
func recordToQuote(r []string) (Quote, bool, error) {
	var (
		quote Quote
		err   error
	)
	for _, item := range r {
		if item == "null" {
			return quote, false, nil
		}
	}
	quote.Date, err = time.Parse("2006-01-02", r[0])
	if err != nil {
		return quote, false, err
	}
	quote.Open, err = strconv.ParseFloat(r[1], 64)
	if err != nil {
		return quote, false, err
	}
	quote.High, err = strconv.ParseFloat(r[2], 64)
	if err != nil {
		return quote, false, err
	}
	quote.Low, err = strconv.ParseFloat(r[3], 64)
	if err != nil {
		return quote, false, err
	}
	quote.Close, err = strconv.ParseFloat(r[4], 64)
	if err != nil {
		return quote, false, err
	}
	quote.AdjClose, err = strconv.ParseFloat(r[5], 64)
	if err != nil {
		return quote, false, err
	}
	quote.Volume, err = strconv.Atoi(r[6])
	if err != nil {
		return quote, false, err
	}
	return quote, true, nil
}
