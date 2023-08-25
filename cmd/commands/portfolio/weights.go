// Copyright 2020 Silvio Böhler
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package portfolio

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/sboehler/knut/cmd/flags"
	"github.com/sboehler/knut/lib/common/date"
	"github.com/sboehler/knut/lib/common/predicate"
	"github.com/sboehler/knut/lib/common/table"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/performance"
	"github.com/sboehler/knut/lib/model"
	"github.com/sboehler/knut/lib/model/registry"
	"github.com/sboehler/knut/lib/reports/weights"
)

// CreateWeightsCommand creates the command.
func CreateWeightsCommand() *cobra.Command {

	var r weightsRunner
	// Cmd is the balance command.
	c := &cobra.Command{
		Use:   "weights",
		Short: "compute portfolio weights",
		Long:  `Compute portfolio weights.`,

		Args: cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),

		Run: r.run,
	}
	r.setupFlags(c)
	return c
}

type weightsRunner struct {
	valuation             flags.CommodityFlag
	accounts, commodities flags.RegexFlag

	// alignment
	period   flags.PeriodFlag
	last     int
	interval flags.IntervalFlags

	// formatting
	thousands bool
	color     bool
	digits    int32

	omitCommodities    bool
	mapping            flags.MappingFlag
	sortAlphabetically bool

	universe string

	csv bool
}

func (r *weightsRunner) setupFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&r.universe, "universe", "", "", "universe file")
	cmd.Flags().VarP(&r.valuation, "val", "v", "valuate in the given commodity")
	cmd.Flags().Var(&r.accounts, "account", "filter accounts with a regex")
	cmd.Flags().Var(&r.commodities, "commodity", "filter commodities with a regex")
	r.period.Setup(cmd, date.Period{End: date.Today()})
	r.interval.Setup(cmd, date.Once)
	cmd.Flags().BoolVarP(&r.sortAlphabetically, "sort", "a", false, "Sort accounts alphabetically")
	cmd.Flags().IntVar(&r.last, "last", 0, "last n periods")
	cmd.Flags().BoolVar(&r.csv, "csv", false, "render csv")
	cmd.Flags().BoolVar(&r.omitCommodities, "omit-commodities", false, "don't render commodities")
	cmd.Flags().VarP(&r.mapping, "map", "m", "<level>,<regex>")
	cmd.Flags().Int32Var(&r.digits, "digits", 0, "round to number of digits")
	cmd.Flags().BoolVarP(&r.thousands, "thousands", "k", false, "show numbers in units of 1000")
	cmd.Flags().BoolVar(&r.color, "color", true, "print output in color")

}

func (r *weightsRunner) run(cmd *cobra.Command, args []string) {
	if err := r.execute(cmd, args); err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), err)
		os.Exit(1)
	}
}

func (r *weightsRunner) execute(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	reg := registry.New()
	var universe performance.Universe
	if len(r.universe) > 0 {
		var err error
		universe, err = performance.LoadUniverseFromFile(reg.Commodities(), r.universe)
		if err != nil {
			return err
		}
	}
	valuation, err := r.valuation.Value(reg)
	if err != nil {
		return err
	}
	j, err := journal.FromPath(ctx, reg, args[0])
	if err != nil {
		return err
	}
	partition := date.NewPartition(r.period.Value().Clip(j.Period()), r.interval.Value(), r.last)
	calculator := &performance.Calculator{
		Context:         reg,
		Valuation:       valuation,
		AccountFilter:   predicate.ByName[*model.Account](r.accounts.Regex()),
		CommodityFilter: predicate.ByName[*model.Commodity](r.commodities.Regex()),
	}
	j.Fill(partition.EndDates()...)
	rep := weights.NewReport()
	_, err = j.Process(
		journal.ComputePrices(valuation),
		journal.Balance(reg, valuation),
		calculator.ComputeValues(),
		weights.Query{
			OmitCommodities: r.omitCommodities,
			Universe:        universe,
			Partition:       partition,
			Mapping:         r.mapping.Value(),
		}.Execute(j, rep),
	)
	if err != nil {
		return err
	}
	reportRenderer := weights.Renderer{
		SortAlphabetically: r.sortAlphabetically,
	}
	var tableRenderer Renderer
	if r.csv {
		tableRenderer = &table.CSVRenderer{}
	} else {
		tableRenderer = &table.TextRenderer{
			Color: r.color,
			Round: r.digits,
		}
	}
	out := bufio.NewWriter(cmd.OutOrStdout())
	defer out.Flush()
	return tableRenderer.Render(reportRenderer.Render(rep), out)
}

type Renderer interface {
	Render(*table.Table, io.Writer) error
}
