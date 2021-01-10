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

package transcode

import (
	"bufio"
	"fmt"

	"github.com/sboehler/knut/lib/balance"
	"github.com/sboehler/knut/lib/beancount"
	"github.com/sboehler/knut/lib/ledger"
	"github.com/sboehler/knut/lib/model/commodities"
	"github.com/sboehler/knut/lib/parser"

	"github.com/spf13/cobra"
	"go.uber.org/multierr"
)

// CreateCmd creates the command.
func CreateCmd() *cobra.Command {

	// Cmd is the balance command.
	c := &cobra.Command{
		Use:   "transcode",
		Short: "transcode to beancount",
		Long: `Transcode the given journal to beancount, to leverage their amazing tooling. This command requires a valuation commodity, so` +
			` that all currency conversions can be done by knut.`,

		Args: cobra.ExactValidArgs(1),

		RunE: run,
	}
	c.Flags().StringP("commodity", "c", "", "valuate in the given commodity")
	return c
}

func run(cmd *cobra.Command, args []string) error {
	c, err := cmd.Flags().GetString("commodity")
	if err != nil {
		return err
	}
	if c == "" {
		return fmt.Errorf("missing --commodity flag, please provide a valuation commodity")
	}
	commodity := commodities.Get(c)

	ch, err := parser.Parse(args[0])
	if err != nil {
		return err
	}
	l, err := ledger.Build(ledger.Options{}, ch)
	if err != nil {
		return err
	}
	if err = process(commodity, l); err != nil {
		return err
	}
	w := bufio.NewWriter(cmd.OutOrStdout())
	defer func() { err = multierr.Append(err, w.Flush()) }()

	// transcode the ledger here
	return beancount.Transcode(w, l, commodity)
}

// process processes the ledger and creates valuations for the given commodities
func process(c *commodities.Commodity, l ledger.Ledger) error {
	balance := balance.New(c)
	for _, day := range l {
		if err := balance.Update(day); err != nil {
			return err
		}
	}
	return nil
}
