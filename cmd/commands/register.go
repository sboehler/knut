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

package commands

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"runtime/pprof"

	"github.com/sboehler/knut/cmd/flags"
	"github.com/sboehler/knut/lib/amounts"
	"github.com/sboehler/knut/lib/common/mapper"
	"github.com/sboehler/knut/lib/common/predicate"
	"github.com/sboehler/knut/lib/common/table"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/check"
	"github.com/sboehler/knut/lib/model"
	"github.com/sboehler/knut/lib/model/account"
	"github.com/sboehler/knut/lib/model/commodity"
	"github.com/sboehler/knut/lib/model/registry"
	"github.com/sboehler/knut/lib/reports/register"

	"github.com/spf13/cobra"
)

// CreateRegisterCmd creates the command.
func CreateRegisterCmd() *cobra.Command {

	var r registerRunner

	// Cmd is the balance command.
	c := &cobra.Command{
		Use:    "register",
		Short:  "create a register sheet",
		Long:   `Compute a register report.`,
		Args:   cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		Run:    r.run,
		Hidden: true,
	}
	r.setupFlags(c)
	return c
}

type registerRunner struct {
	flags.Multiperiod

	// internal
	cpuprofile string

	// transformations
	showCommodities               bool
	showSource                    bool
	showDescriptions              bool
	mapping                       flags.MappingFlag
	remap                         flags.RegexFlag
	valuation                     flags.CommodityFlag
	accounts, others, commodities flags.RegexFlag

	// formatting
	thousands, color   bool
	sortAlphabetically bool
	digits             int32
}

func (r *registerRunner) run(cmd *cobra.Command, args []string) {
	if r.cpuprofile != "" {
		f, err := os.Create(r.cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	if err := r.execute(cmd, args); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "%+v\n", err)
		os.Exit(1)
	}
}

func (r *registerRunner) setupFlags(c *cobra.Command) {
	r.Multiperiod.Setup(c)
	c.Flags().StringVar(&r.cpuprofile, "cpuprofile", "", "file to write profile")
	c.Flags().BoolVarP(&r.sortAlphabetically, "sort", "s", false, "Sort accounts alphabetically")
	c.Flags().BoolVarP(&r.showCommodities, "show-commodities", "c", false, "Show commodities")
	c.Flags().BoolVarP(&r.showDescriptions, "show-descriptions", "d", false, "Show descriptions")
	c.Flags().BoolVarP(&r.showSource, "show-source", "a", false, "Show the source accounts")
	c.Flags().VarP(&r.valuation, "val", "v", "valuate in the given commodity")
	c.Flags().VarP(&r.mapping, "map", "m", "<level>,<regex>")
	c.Flags().VarP(&r.remap, "remap", "r", "<regex>")
	c.Flags().Var(&r.accounts, "source", "filter source accounts with a regex")
	c.Flags().Var(&r.others, "dest", "filter dest accounts with a regex")
	c.Flags().Var(&r.commodities, "commodity", "filter commodities with a regex")
	c.Flags().Int32Var(&r.digits, "digits", 0, "round to number of digits")
	c.Flags().BoolVarP(&r.thousands, "thousands", "k", false, "show numbers in units of 1000")
	c.Flags().BoolVar(&r.color, "color", true, "print output in color")
}

func (r registerRunner) execute(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	reg := registry.New()
	valuation, err := r.valuation.Value(reg)
	if err != nil {
		return err
	}
	r.showCommodities = r.showCommodities || valuation == nil
	j, err := journal.FromPath(ctx, reg, args[0])
	if err != nil {
		return err
	}
	var am mapper.Mapper[*model.Account]
	if r.showSource {
		am = account.Remap(reg.Accounts(), r.remap.Regex())
	}
	partition := r.Multiperiod.Partition(j.Period())
	rep := register.NewReport(reg)
	_, err = j.Process(
		journal.Sort(),
		journal.ComputePrices(valuation),
		check.Check(),
		journal.Valuate(reg, valuation),
		journal.Filter(partition),
		journal.Query{
			Select: amounts.KeyMapper{
				Date:    partition.Align(),
				Account: am,
				Other: mapper.Sequence(
					account.Remap(reg.Accounts(), r.remap.Regex()),
					account.Shorten(reg.Accounts(), r.mapping.Value()),
				),
				Commodity:   commodity.IdentityIf(r.showCommodities),
				Valuation:   mapper.Identity[*commodity.Commodity],
				Description: mapper.IdentityIf[string](r.showDescriptions),
			}.Build(),
			Where: predicate.And(
				amounts.AccountMatches(r.accounts.Regex()),
				amounts.OtherAccountMatches(r.others.Regex()),
				amounts.CommodityMatches(r.commodities.Regex()),
			),
			Valuation: valuation,
		}.Into(rep),
	)
	if err != nil {
		return err
	}
	reportRenderer := register.Renderer{
		ShowCommodities:    r.showCommodities,
		ShowDescriptions:   r.showDescriptions,
		ShowSource:         r.showSource,
		SortAlphabetically: r.sortAlphabetically,
	}
	tableRenderer := table.TextRenderer{
		Color:     r.color,
		Thousands: r.thousands,
		Round:     r.digits,
	}
	out := bufio.NewWriter(cmd.OutOrStdout())
	defer out.Flush()
	return tableRenderer.Render(reportRenderer.Render(rep), out)
}
