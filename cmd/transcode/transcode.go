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

	"github.com/sboehler/knut/lib/balance"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
	"github.com/sboehler/knut/lib/journal/ast/beancount"
	"github.com/sboehler/knut/lib/journal/ast/parser"

	"github.com/spf13/cobra"
	"go.uber.org/multierr"
)

// CreateCmd creates the command.
func CreateCmd() *cobra.Command {
	// Cmd is the balance command.
	var cmd = &cobra.Command{
		Use:   "transcode",
		Short: "transcode to beancount",
		Long: `Transcode the given journal to beancount, to leverage their amazing tooling. This command requires a valuation commodity, so` +
			` that all currency conversions can be done by knut.`,

		Args: cobra.ExactValidArgs(1),

		Run: run,
	}
	cmd.Flags().StringP("commodity", "c", "", "valuate in the given commodity")
	return cmd
}

func run(cmd *cobra.Command, args []string) {
	if err := execute(cmd, args); err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), err)
		os.Exit(1)
	}
}

func execute(cmd *cobra.Command, args []string) (errors error) {
	c, err := cmd.Flags().GetString("commodity")
	if err != nil {
		return err
	}
	if c == "" {
		return fmt.Errorf("missing --commodity flag, please provide a valuation commodity")
	}
	var (
		ctx       = journal.NewContext()
		commodity *journal.Commodity
		j         = parser.RecursiveParser{Context: ctx, File: args[0]}
		l         *ast.AST
	)
	if commodity, err = ctx.GetCommodity(c); err != nil {
		return err
	}
	if l, err = ast.FromDirectives(ctx, journal.Filter{}, j.Parse()); err != nil {
		return err
	}
	var (
		bal   = balance.New(ctx, commodity)
		steps = []ast.Processor{
			balance.DateUpdater{Balance: bal},
			balance.AccountOpener{Balance: bal},
			balance.TransactionBooker{Balance: bal},
			balance.ValueBooker{Balance: bal},
			balance.Asserter{Balance: bal},
			&balance.PriceUpdater{Balance: bal},
			balance.TransactionValuator{Balance: bal},
			balance.ValuationTransactionComputer{Balance: bal},
			balance.AccountCloser{Balance: bal},
		}
	)
	if err := l.Process(steps); err != nil {
		return err
	}
	var w = bufio.NewWriter(cmd.OutOrStdout())
	defer func() { err = multierr.Append(err, w.Flush()) }()

	// transcode the ledger here
	return beancount.Transcode(w, l, commodity)
}
