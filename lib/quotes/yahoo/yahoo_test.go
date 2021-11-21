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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestFetch(t *testing.T) {
	var (
		gotQuery map[string][]string
		response = "Date,Open,High,Low,Close,Adj Close,Volume\n" +
			"2019-11-07,1294.280029,1323.739990,1294.244995,1308.859985,1308.859985,2030000\n" +
			"2019-11-08,1305.280029,1318.000000,1304.364990,1311.369995,1311.369995,1251400"
		srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotQuery = r.URL.Query()
			w.Write([]byte(response))
		}))
	)
	defer srv.Close()
	var (
		want = []Quote{
			{
				Date:     time.Date(2019, 11, 07, 0, 0, 0, 0, time.UTC),
				Open:     1294.280029,
				High:     1323.73999,
				Low:      1294.244995,
				Close:    1308.859985,
				AdjClose: 1308.859985,
				Volume:   2030000,
			},
			{
				Date:     time.Date(2019, 11, 8, 0, 0, 0, 0, time.UTC),
				Open:     1305.280029,
				High:     1318,
				Low:      1304.36499,
				Close:    1311.369995,
				AdjClose: 1311.369995,
				Volume:   1251400,
			},
		}
		wantQuery = map[string][]string{
			"period1":  {"1573084800"},
			"period2":  {"1573257600"},
			"events":   {"history"},
			"interval": {"1d"},
		}
		client = Client{srv.URL}
	)

	got, err := client.Fetch("GOOG", time.Date(2019, 11, 7, 0, 0, 0, 0, time.UTC), time.Date(2019, 11, 9, 0, 0, 0, 0, time.UTC))

	if diff := cmp.Diff(wantQuery, gotQuery); diff != "" {
		t.Errorf("client.Fetch(): unexpected diff in query parameters (-want, +got):\n%s", diff)
	}
	if err != nil {
		t.Errorf("client.Fetch(): returned unexpected error %v", err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("client.Fetch() returned difference (-want, +got):\n%s", diff)
	}
}
