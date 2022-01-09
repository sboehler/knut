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
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/sboehler/knut/cmd/flags"
	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast/beancount"
	"github.com/sboehler/knut/lib/journal/ast/parser"
	"github.com/sboehler/knut/lib/journal/process"
	"github.com/sboehler/knut/lib/journal/val"

	"github.com/spf13/cobra"
	"go.uber.org/multierr"
)

// CreateCmd creates the command.
func CreateCmd() *cobra.Command {
	var r runner

	// Cmd is the balance command.
	var cmd = &cobra.Command{
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
	var (
		par = parser.RecursiveParser{
			Context: jctx,
			File:    args[0],
		}
		astBuilder = process.ASTBuilder{
			Context: jctx,
		}
		pastBuilder = process.PASTBuilder{
			Context: jctx,
			Expand:  true,
		}
		priceUpdater = process.PriceUpdater{
			Context:   jctx,
			Valuation: valuation,
		}
		valuator = process.Valuator{
			Context:   jctx,
			Valuation: valuation,
		}
	)

	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	ch0, errCh0 := par.Parse(ctx)
	ch1, errCh1 := astBuilder.BuildAST(ctx, ch0)
	ch2, errCh2 := pastBuilder.ProcessAST(ctx, ch1)
	ch3, errCh3 := priceUpdater.ProcessStream(ctx, ch2)
	resCh, errCh4 := valuator.ProcessStream(ctx, ch3)

	errCh := mergeErrors(errCh0, errCh1, errCh2, errCh3, errCh4)

	var days []*val.Day
	for {
		d, ok, err := cpr.Get(resCh, errCh)
		if !ok {
			break
		}
		if err != nil {
			return err
		}
		days = append(days, d)
	}

	var w = bufio.NewWriter(cmd.OutOrStdout())
	defer func() { err = multierr.Append(err, w.Flush()) }()

	// transcode the ledger here
	return beancount.Transcode(w, days, valuation)
}

func mergeErrors(inChs ...<-chan error) chan error {
	var (
		wg    sync.WaitGroup
		errCh = make(chan error)
	)
	wg.Add(len(inChs))
	for _, inCh := range inChs {
		go func(ch <-chan error) {
			defer wg.Done()
			for err := range ch {
				errCh <- err
			}
		}(inCh)
	}
	go func() {
		wg.Wait()
		close(errCh)
	}()
	return errCh
}
