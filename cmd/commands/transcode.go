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
	"os"

	"github.com/sboehler/knut/cmd/flags"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/beancount"
	"github.com/sboehler/knut/lib/model"
	"github.com/sboehler/knut/lib/model/registry"

	"github.com/spf13/cobra"
	"go.uber.org/multierr"
)

// CreateTranscodeCommand creates the command.
func CreateTranscodeCommand() *cobra.Command {
	var r transcodeRunner

	// Cmd is the balance command.
	cmd := &cobra.Command{
		Use:   "transcode",
		Short: "transcode to beancount",
		Long: `Transcode the given journal to beancount, to leverage their amazing tooling. This command requires a valuation commodity, so` +
			` that all currency conversions can be done by knut.`,

		Args: cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),

		Run: r.run,
	}
	r.setupFlags(cmd)
	return cmd
}

type transcodeRunner struct {
	valuation flags.CommodityFlag
}

func (r *transcodeRunner) setupFlags(c *cobra.Command) {
	c.Flags().VarP(&r.valuation, "val", "v", "valuate in the given commodity")
}

func (r *transcodeRunner) run(cmd *cobra.Command, args []string) {
	if err := r.execute(cmd, args); err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), err)
		os.Exit(1)
	}
}

func (r *transcodeRunner) execute(cmd *cobra.Command, args []string) (errors error) {
	var (
		reg       = registry.New()
		valuation *model.Commodity
		err       error
	)
	if valuation, err = r.valuation.Value(reg); err != nil {
		return err
	}
	j, err := journal.FromPath(cmd.Context(), reg, args[0])
	if err != nil {
		return err
	}
	ds, err := j.Process(
		journal.Sort(),
		journal.ComputePrices(valuation),
		journal.Balance(reg, valuation),
	)
	w := bufio.NewWriter(cmd.OutOrStdout())
	defer func() { err = multierr.Append(err, w.Flush()) }()

	return beancount.Transcode(w, ds, valuation)
}
