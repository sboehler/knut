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

package prices

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
	"github.com/sboehler/knut/lib/journal/ast/parser"
	"github.com/sboehler/knut/lib/journal/ast/printer"
	"github.com/sboehler/knut/lib/quotes/yahoo"
	"github.com/shopspring/decimal"
	"go.uber.org/multierr"

	"github.com/cheggaaa/pb/v3"
	"github.com/natefinch/atomic"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

// CreateCmd creates the command.
func CreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fetch",
		Short: "Fetch quotes from Yahoo! Finance",
		Long:  `Fetch quotes from Yahoo! Finance based on the supplied configuration in yaml format. See doc/prices.yaml for an example.`,

		Args: cobra.ExactValidArgs(1),

		Run: run,
	}
}

func run(cmd *cobra.Command, args []string) {
	if err := execute(cmd, args); err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), err)
		os.Exit(1)
	}
}

const concurrency = 5

func execute(cmd *cobra.Command, args []string) (errors error) {
	var ctx = journal.NewContext()
	configs, err := readConfig(args[0])
	if err != nil {
		return err
	}
	var errCh = make(chan error)
	defer close(errCh)
	go func() {
		for err = range errCh {
			errors = multierr.Append(err, errors)
		}
	}()
	var sema = make(chan bool, concurrency)
	defer close(sema)
	var bar = pb.StartNew(len(configs))
	defer bar.Finish()
	for _, cfg := range configs {
		sema <- true
		go func(c config) {
			if err := fetch(ctx, args[0], c); err != nil {
				errCh <- err
			}
			bar.Increment()
			<-sema
		}(cfg)
	}
	for i := 0; i < concurrency; i++ {
		sema <- true
	}
	return errors
}

func fetch(ctx journal.Context, f string, cfg config) error {
	var absPath = filepath.Join(filepath.Dir(f), cfg.File)
	l, err := readFile(ctx, absPath)
	if err != nil {
		return err
	}
	if err := fetchPrices(ctx, cfg, time.Now().AddDate(-1, 0, 0), time.Now(), l); err != nil {
		return err
	}
	if err := writeFile(ctx, l, absPath); err != nil {
		return err
	}
	return nil
}

func readConfig(path string) ([]config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var dec = yaml.NewDecoder(f)
	dec.SetStrict(true)
	var t []config
	if err := dec.Decode(&t); err != nil {
		return nil, err
	}
	return t, nil
}

func readFile(ctx journal.Context, filepath string) (res map[time.Time]*ast.Price, err error) {
	p, cls, err := parser.FromPath(ctx, filepath)
	if err != nil {
		return nil, err
	}
	defer func() { err = multierr.Append(err, cls()) }()
	var prices = make(map[time.Time]*ast.Price)
	for i := range p.ParseAll() {
		switch d := i.(type) {
		case error:
			return nil, err
		case *ast.Price:
			prices[d.Date] = d
		default:
			return nil, fmt.Errorf("unexpected directive in prices file: %v", d)
		}
	}
	return prices, nil
}

func fetchPrices(ctx journal.Context, cfg config, t0, t1 time.Time, results map[time.Time]*ast.Price) error {
	var (
		c                 = yahoo.New()
		quotes            []yahoo.Quote
		commodity, target *journal.Commodity
		err               error
	)
	if quotes, err = c.Fetch(cfg.Symbol, t0, t1); err != nil {
		return err
	}
	if commodity, err = ctx.GetCommodity(cfg.Commodity); err != nil {
		return err
	}
	if target, err = ctx.GetCommodity(cfg.TargetCommodity); err != nil {
		return err
	}
	for _, i := range quotes {
		results[i.Date] = &ast.Price{
			Date:      i.Date,
			Commodity: commodity,
			Target:    target,
			Price:     decimal.NewFromFloat(i.Close),
		}
	}
	return nil
}

func writeFile(ctx journal.Context, prices map[time.Time]*ast.Price, filepath string) error {
	var b = ast.NewBuilder(ctx, journal.Filter{})
	for _, price := range prices {
		b.AddPrice(price)
	}
	r, w := io.Pipe()
	go func() {
		defer w.Close()
		_, err := printer.New().PrintLedger(w, b.Build())
		if err != nil {
			panic(err)
		}
	}()
	return atomic.WriteFile(filepath, r)
}

type config struct {
	Symbol          string `yaml:"symbol"`
	File            string `yaml:"file"`
	Commodity       string `yaml:"commodity"`
	TargetCommodity string `yaml:"target_commodity"`
}
