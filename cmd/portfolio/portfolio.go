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

package portfolio

import (
	"fmt"
	"log"
	"os"
	"runtime/pprof"

	"github.com/spf13/cobra"

	"github.com/sboehler/knut/cmd/flags"
	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast/parser"
	"github.com/sboehler/knut/lib/journal/process"
	"github.com/sboehler/knut/lib/journal/process/performance"
)

// CreateCmd creates the command.
func CreateCmd() *cobra.Command {

	var r runner
	// Cmd is the balance command.
	var c = &cobra.Command{
		Use:   "portfolio",
		Short: "compute portfolio returns",
		Long:  `Compute portfolio returns.`,

		Args: cobra.ExactValidArgs(1),

		Hidden: true,

		Run: r.run,
	}
	r.setupFlags(c)
	return c
}

type runner struct {
	cpuprofile            string
	valuation             flags.CommodityFlag
	accounts, commodities flags.RegexFlag
}

func (r *runner) setupFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&r.cpuprofile, "cpuprofile", "", "file to write profile")
	cmd.Flags().VarP(&r.valuation, "val", "v", "valuate in the given commodity")
	cmd.Flags().Var(&r.accounts, "account", "filter accounts with a regex")
	cmd.Flags().Var(&r.commodities, "commodity", "filter commodities with a regex")
}

func (r *runner) run(cmd *cobra.Command, args []string) {
	if r.cpuprofile != "" {
		f, err := os.Create(r.cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	if err := r.execute(cmd, args); err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), err)
		os.Exit(1)
	}
}

func (r *runner) execute(cmd *cobra.Command, args []string) error {
	var (
		ctx       = cmd.Context()
		jctx      = journal.NewContext()
		valuation *journal.Commodity
		err       error
	)
	if valuation, err = r.valuation.Value(jctx); err != nil {
		return err
	}

	var (
		par = parser.RecursiveParser{
			File:    args[0],
			Context: jctx,
		}
		astBuilder = process.ASTBuilder{
			Context: jctx,
		}
		astExpander = process.ASTExpander{
			Expand: true,
		}
		pastBuilder = process.PASTBuilder{
			Context: jctx,
		}
		priceUpdater = process.PriceUpdater{
			Context:   jctx,
			Valuation: valuation,
		}
		valuator = process.Valuator{
			Context:   jctx,
			Valuation: valuation,
		}
		calculator = performance.Calculator{
			Context:   jctx,
			Valuation: valuation,
			Filter: journal.Filter{
				Accounts:    r.accounts.Value(),
				Commodities: r.commodities.Value(),
			},
		}
	)

	ch0, errCh0 := par.Parse(ctx)
	ch1, errCh1 := astBuilder.BuildAST(ctx, ch0)
	ch2, errCh2 := astExpander.ExpandAndFilterAST(ctx, ch1)
	ch3, errCh3 := pastBuilder.ProcessAST(ctx, ch2)
	ch4, errCh4 := priceUpdater.ProcessStream(ctx, ch3)
	ch5, errCh5 := valuator.ProcessStream(ctx, ch4)
	resCh, errCh6 := calculator.Perf(ctx, ch5)

	errCh := cpr.Demultiplex(errCh0, errCh1, errCh2, errCh3, errCh4, errCh5, errCh6)

	for {
		p, ok, err := cpr.Get(resCh, errCh)
		if !ok || err != nil {
			return err
		}
		fmt.Printf("%v: %.1f%%\n", p.Date.Format("2006-01-02"), 100*(p.Performance()-1))
	}
}
