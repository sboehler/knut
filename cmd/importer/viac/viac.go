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

package viac

import (
	"bufio"
	"encoding/json"
	"io/ioutil"
	"time"

	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"

	"github.com/sboehler/knut/cmd/flags"
	"github.com/sboehler/knut/cmd/importer"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/printer"
)

// CreateCmd creates the command.
func CreateCmd() *cobra.Command {
	var r runner
	var cmd = &cobra.Command{
		Use:   "ch.viac",
		Short: "Import VIAC values from JSON files",
		Long:  `Open app.viac.ch, choose a portfolio, and select "From start" in the overview dash. In the Chrome dev tools, save the response from the "performance" XHR call, and pass the resulting file to this importer.`,

		Args: cobra.ExactValidArgs(1),

		RunE: r.run,
	}
	r.setupFlags(cmd)
	return cmd
}

func init() {
	importer.Register(CreateCmd)
}

func (r *runner) setupFlags(cmd *cobra.Command) {
	cmd.Flags().VarP(&r.from, "from", "f", "YYYY-MM-DD - ignore entries before this date")
	cmd.Flags().VarP(&r.account, "account", "a", "account name")
}

type runner struct {
	from    flags.DateFlag
	account flags.AccountFlag
}

func (r *runner) run(cmd *cobra.Command, args []string) error {
	var (
		ctx     = journal.NewContext()
		f       *bufio.Reader
		account *journal.Account
		err     error
	)

	if account, err = r.account.Value(ctx); err != nil {
		return err
	}
	commodity, err := ctx.GetCommodity("CHF")
	if err != nil {
		return err
	}
	if f, err = flags.OpenFile(args[0]); err != nil {
		return err
	}
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}
	var resp response
	json.Unmarshal(b, &resp)

	var builder = journal.New(ctx)
	for _, dv := range resp.DailyValues {
		d, err := time.Parse("2006-01-02", dv.Date)
		if err != nil {
			return err
		}
		if d.Before(r.from.Value()) {
			continue
		}
		a, err := decimal.NewFromString(dv.Value.String())
		if err != nil {
			return err
		}
		builder.AddValue(&journal.Value{
			Date:      d,
			Account:   account,
			Amount:    a.Round(2),
			Commodity: commodity,
		})
	}

	out := bufio.NewWriter(cmd.OutOrStdout())
	defer out.Flush()
	_, err = printer.New().PrintLedger(out, builder.SortedDays())
	return err
}

type response struct {
	DailyValues []dailyValue `json:"dailyValues"`
}

type dailyValue struct {
	Date  string      `json:"date"`
	Value json.Number `json:"value"`
}
