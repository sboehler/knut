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

package asyncbalance

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"runtime/pprof"
	"time"

	"github.com/sboehler/knut/cmd/flags"
	"github.com/sboehler/knut/lib/balance"
	"github.com/sboehler/knut/lib/ledger"
	"github.com/sboehler/knut/lib/parser"
	"github.com/sboehler/knut/lib/report"
	"github.com/sboehler/knut/lib/table"

	"github.com/spf13/cobra"
)

// CreateCmd creates the command.
func CreateCmd() *cobra.Command {

	// Cmd is the balance command.
	var c = &cobra.Command{
		Use:   "asyncbalance",
		Short: "create a balance sheet",
		Long:  `Compute a balance for a date or set of dates.`,

		Args: cobra.ExactValidArgs(1),

		Hidden: true,

		Run: run,
	}
	c.Flags().String("from", "", "from date")
	c.Flags().String("cpuprofile", "", "file to write profile")
	c.Flags().String("to", "", "to date")
	c.Flags().IntP("last", "l", 0, "last n periods")
	c.Flags().BoolP("diff", "d", false, "diff")
	c.Flags().BoolP("show-commodities", "s", false, "Show commodities on their own rows")
	c.Flags().Bool("days", false, "days")
	c.Flags().Bool("weeks", false, "weeks")
	c.Flags().Bool("months", false, "months")
	c.Flags().Bool("quarters", false, "quarters")
	c.Flags().Bool("years", false, "years")
	c.Flags().StringP("val", "v", "", "valuate in the given commodity")
	c.Flags().StringArrayP("collapse", "c", nil, "<level>,<regex>")
	c.Flags().String("account", "", "filter accounts with a regex")
	c.Flags().String("commodity", "", "filter commodities with a regex")
	c.Flags().Int32("digits", 0, "round to number of digits")
	c.Flags().BoolP("thousands", "k", false, "show numbers in units of 1000")
	c.Flags().Bool("color", false, "print output in color")
	return c
}

func run(cmd *cobra.Command, args []string) {
	if err := execute(cmd, args); err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), err)
		os.Exit(1)
	}
}

func execute(cmd *cobra.Command, args []string) error {
	prof, err := cmd.Flags().GetString("cpuprofile")
	if err != nil {
		return err
	}
	if prof != "" {
		f, err := os.Create(prof)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	var out = bufio.NewWriter(cmd.OutOrStdout())
	defer out.Flush()
	return process(cmd, args, out)
}

func process(cmd *cobra.Command, args []string, w io.Writer) error {
	var (
		ctx      = ledger.NewContext()
		from, to *time.Time
		err      error
	)
	if cmd.Flags().Changed("from") {
		if from, err = flags.GetDateFlag(cmd, "from"); err != nil {
			return err
		}
	}
	if cmd.Flags().Changed("to") {
		if to, err = flags.GetDateFlag(cmd, "to"); err != nil {
			return err
		}
	} else {
		now := time.Now()
		d := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		to = &d
	}
	last, err := cmd.Flags().GetInt("last")
	if err != nil {
		return err
	}
	var valuation *ledger.Commodity
	if cmd.Flags().Changed("val") {
		if valuation, err = flags.GetCommodityFlag(cmd, ctx, "val"); err != nil {
			return err
		}
	}
	showCommodities, err := cmd.Flags().GetBool("show-commodities")
	if err != nil {
		return err
	}
	diff, err := cmd.Flags().GetBool("diff")
	if err != nil {
		return err
	}
	digits, err := cmd.Flags().GetInt32("digits")
	if err != nil {
		return err
	}
	period, err := flags.GetPeriodFlag(cmd)
	if err != nil {
		return err
	}
	mapping, err := flags.GetCollapseFlag(cmd, "collapse")
	if err != nil {
		return err
	}
	filterAccounts, err := flags.GetRegexFlag(cmd, "account")
	if err != nil {
		return err
	}
	filterCommodities, err := flags.GetRegexFlag(cmd, "commodity")
	if err != nil {
		return err
	}
	thousands, err := cmd.Flags().GetBool("thousands")
	if err != nil {
		return err
	}
	color, err := cmd.Flags().GetBool("color")
	if err != nil {
		return err
	}

	var (
		parser = parser.RecursiveParser{
			File:    args[0],
			Context: ctx,
		}
		bal    = balance.New(ctx, valuation)
		filter = ledger.Filter{
			Accounts:    filterAccounts,
			Commodities: filterCommodities,
		}
		rep = &report.Report{
			Value:   valuation != nil,
			Mapping: mapping,
		}
		reportRenderer = report.Renderer{
			Context:         ctx,
			ShowCommodities: showCommodities || valuation == nil,
			Report:          rep,
		}
		tableRenderer = table.TextRenderer{
			Color:     color,
			Thousands: thousands,
			Round:     digits,
		}
	)

	var l ledger.Ledger
	if l, err = parser.BuildLedger(filter); err != nil {
		return err
	}

	var (
		balCh        = make(chan *balance.Balance)
		cotx, cancel = context.WithCancel(context.Background())
	)
	defer cancel()

	b1 := balance.SetDate(cotx, l, balCh)
	b2 := balance.UpdatePrices(cotx, l, valuation, b1)
	b3, snapshots := balance.Snapshot(cotx, balance.SnapshotConfig{
		Last:   last,
		From:   from,
		Diff:   diff,
		Period: period,
		To:     to,
	}, l, b2)

	go func() {
		defer close(balCh)
		for range l.Days {
			balCh <- bal
			<-b3
		}
	}()

	for bal := range snapshots {
		rep.Add(bal)
	}
	return tableRenderer.Render(reportRenderer.Render(), w)
}
