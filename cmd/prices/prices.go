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

package prices

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/sboehler/knut/lib/ledger"
	"github.com/sboehler/knut/lib/model/commodities"
	"github.com/sboehler/knut/lib/parser"
	"github.com/sboehler/knut/lib/printer"
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

func execute(cmd *cobra.Command, args []string) (errors error) {
	configs, err := readConfig(args[0])
	if err != nil {
		return err
	}
	errCh := make(chan error)
	defer close(errCh)
	go func() {
		for err = range errCh {
			errors = multierr.Append(err, errors)
		}
	}()
	concurrency := 5
	sema := make(chan bool, concurrency)
	defer close(sema)
	bar := pb.StartNew(len(configs))
	defer bar.Finish()
	for _, cfg := range configs {
		sema <- true
		go func(c config) {
			if err := fetch(args[0], c); err != nil {
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

func fetch(f string, cfg config) error {
	absPath := filepath.Join(filepath.Dir(f), cfg.File)
	l, err := readFile(absPath)
	if err != nil {
		return err
	}
	if err := fetchPrices(cfg, time.Now().AddDate(-1, 0, 0), time.Now(), l); err != nil {
		return err
	}
	if err := writeFile(l, absPath); err != nil {
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
	t := []config{}
	dec := yaml.NewDecoder(f)
	dec.SetStrict(true)
	if err := dec.Decode(&t); err != nil {
		return nil, err
	}
	return t, nil
}

func readFile(filepath string) (map[time.Time]*ledger.Price, error) {
	p, err := parser.Open(filepath)
	if err != nil {
		return nil, err
	}
	prices := map[time.Time]*ledger.Price{}
	for {
		d, err := p.Next()
		if err == io.EOF {
			return prices, nil
		}
		if err != nil {
			return nil, err
		}
		if price, ok := d.(*ledger.Price); ok {
			prices[price.Date] = price
		} else {
			return nil, fmt.Errorf("Unexpected directive in prices file: %v", d)
		}
	}
}

func fetchPrices(cfg config, t0, t1 time.Time, results map[time.Time]*ledger.Price) error {
	c := yahoo.New()
	quotes, err := c.Fetch(cfg.Symbol, t0, t1)
	if err != nil {
		return err
	}
	for _, i := range quotes {
		results[i.Date] = &ledger.Price{
			Date:      i.Date,
			Commodity: commodities.Get(cfg.Commodity),
			Target:    commodities.Get(cfg.TargetCommodity),
			Price:     decimal.NewFromFloat(i.Close),
		}
	}
	return nil
}

func writeFile(prices map[time.Time]*ledger.Price, filepath string) error {
	b := ledger.NewBuilder(ledger.Filter{})
	for _, price := range prices {
		b.AddPrice(price)
	}
	r, w := io.Pipe()
	go func() {
		defer w.Close()
		_, err := printer.Printer{}.PrintLedger(w, b.Build())
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
