// Copyright 2020 Silvio Böhler
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

package viac

import (
	"bufio"
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"

	"github.com/sboehler/knut/cmd/importer"
	"github.com/sboehler/knut/lib/ledger"
	"github.com/sboehler/knut/lib/model/commodities"
	"github.com/sboehler/knut/lib/printer"
	"github.com/sboehler/knut/lib/scanner"
)

// CreateCmd creates the command.
func CreateCmd() *cobra.Command {
	var cmd = cobra.Command{
		Use:   "ch.viac",
		Short: "Import VIAC values from JSON files",
		Long:  `Open app.viac.ch, choose a portfolio, and select "From start" in the overview dash. In the Chrome dev tools, save the response from the "performance" XHR call, and pass the resulting file to this importer.`,

		Args: cobra.ExactValidArgs(1),

		RunE: run,
	}
	cmd.Flags().StringP("from", "f", "0001-01-01", "YYYY-MM-DD - ignore entries before this date")
	cmd.Flags().StringP("account", "a", "", "account name")
	return &cmd
}

func init() {
	importer.Register(CreateCmd)
}

func run(cmd *cobra.Command, args []string) error {
	dateString, err := cmd.Flags().GetString("from")
	if err != nil {
		return err
	}
	fromDate, err := time.Parse("2006-01-02", dateString)
	if err != nil {
		return err
	}
	accountName, err := cmd.Flags().GetString("account")
	if err != nil {
		return err
	}
	s, err := scanner.New(strings.NewReader(accountName), "")
	if err != nil {
		return err
	}
	account, err := scanner.ParseAccount(s)
	if err != nil {
		return err
	}
	f, err := os.Open(args[0])
	if err != nil {
		return err
	}
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	var resp response
	json.Unmarshal(b, &resp)

	var builder = ledger.NewBuilder(ledger.Filter{})
	for _, dv := range resp.DailyValues {
		d, err := time.Parse("2006-01-02", dv.Date)
		if err != nil {
			return err
		}
		if d.Before(fromDate) {
			continue
		}
		a, err := decimal.NewFromString(dv.Value.String())
		if err != nil {
			return err
		}
		builder.AddValue(&ledger.Value{
			Date:      d,
			Account:   account,
			Amount:    a.Round(2),
			Commodity: commodities.Get("CHF"),
		})
	}

	w := bufio.NewWriter(cmd.OutOrStdout())
	defer w.Flush()
	_, err = printer.Printer{}.PrintLedger(w, builder.Build())
	return err
}

type response struct {
	DailyValues []dailyValue `json:"dailyValues"`
}

type dailyValue struct {
	Date  string      `json:"date"`
	Value json.Number `json:"value"`
}
