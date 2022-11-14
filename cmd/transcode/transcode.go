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

package transcode

import (
	"bufio"
	"fmt"
	"os"

	"github.com/sboehler/knut/cmd/flags"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/beancount"

	"github.com/spf13/cobra"
	"go.uber.org/multierr"
)

// CreateCmd creates the command.
func CreateCmd() *cobra.Command {
	var r runner

	// Cmd is the balance command.
	cmd := &cobra.Command{
		Use:   "transcode",
		Short: "transcode to beancount",
		Long: `Transcode the given journal to beancount, to leverage their amazing tooling. This command requires a valuation commodity, so` +
			` that all currency conversions can be done by knut.`,

		Args: cobra.ExactValidArgs(1),

		Run: r.run,
	}
	r.setupFlags(cmd)
	return cmd
}

type runner struct {
	valuation flags.CommodityFlag
}

func (r *runner) setupFlags(c *cobra.Command) {
	c.Flags().VarP(&r.valuation, "val", "v", "valuate in the given commodity")
}

func (r *runner) run(cmd *cobra.Command, args []string) {
	if err := r.execute(cmd, args); err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), err)
		os.Exit(1)
	}
}

func (r *runner) execute(cmd *cobra.Command, args []string) (errors error) {
	var (
		jctx      = journal.NewContext()
		valuation *journal.Commodity
		err       error
	)
	if valuation, err = r.valuation.Value(jctx); err != nil {
		return err
	}
	j, err := journal.FromPath(cmd.Context(), jctx, args[0])
	if err != nil {
		return err
	}
	l, err := j.Process(
		journal.ComputePrices(valuation),
		journal.Balance(jctx, valuation),
	)
	w := bufio.NewWriter(cmd.OutOrStdout())
	defer func() { err = multierr.Append(err, w.Flush()) }()

	// transcode the ledger here
	return beancount.Transcode(w, l.Days, valuation)
}
