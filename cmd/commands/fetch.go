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
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/model"
	"github.com/sboehler/knut/lib/model/price"
	"github.com/sboehler/knut/lib/model/registry"
	"github.com/sboehler/knut/lib/quotes/yahoo"
	"github.com/sboehler/knut/lib/syntax"
	"github.com/shopspring/decimal"
	"github.com/sourcegraph/conc/pool"
	"go.uber.org/multierr"

	"github.com/cheggaaa/pb/v3"
	"github.com/natefinch/atomic"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

// CreateFetchCommand creates the command.
func CreateFetchCommand() *cobra.Command {
	var runner fetchRunner
	return &cobra.Command{
		Use:   "fetch",
		Short: "Fetch quotes from Yahoo! Finance",
		Long:  `Fetch quotes from Yahoo! Finance based on the supplied configuration in yaml format. See doc/prices.yaml for an example.`,

		Args: cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),

		Run: runner.run,
	}
}

type fetchRunner struct{}

func (r *fetchRunner) run(cmd *cobra.Command, args []string) {
	if err := r.execute(cmd, args); err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), err)
		os.Exit(1)
	}
}

const fetchConcurrency = 5

func (r *fetchRunner) execute(cmd *cobra.Command, args []string) error {
	reg := registry.New()
	configs, err := r.readConfig(args[0])
	if err != nil {
		return err
	}
	p := pool.New().WithMaxGoroutines(fetchConcurrency).WithErrors()
	bar := pb.StartNew(len(configs))

	for _, cfg := range configs {
		cfg := cfg
		p.Go(func() error {
			defer bar.Increment()
			return r.fetch(reg, args[0], cfg)
		})
	}
	return multierr.Combine(p.Wait())
}

func (r *fetchRunner) fetch(reg *registry.Registry, f string, cfg fetchConfig) error {
	absPath := filepath.Join(filepath.Dir(f), cfg.File)
	pricesByDate, err := r.readFile(reg, absPath)
	if err != nil {
		return err
	}
	if err := r.fetchPrices(reg, cfg, time.Now().AddDate(-1, 0, 0), time.Now(), pricesByDate); err != nil {
		return err
	}
	if err := r.writeFile(reg, pricesByDate, absPath); err != nil {
		return err
	}
	return nil
}

func (r *fetchRunner) readConfig(path string) ([]fetchConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	dec := yaml.NewDecoder(f)
	dec.SetStrict(true)
	var t []fetchConfig
	if err := dec.Decode(&t); err != nil {
		return nil, err
	}
	return t, nil
}

func (r *fetchRunner) readFile(ctx *registry.Registry, filepath string) (res map[time.Time]*model.Price, err error) {
	f, err := syntax.ParseFile(filepath)
	if err != nil {
		return nil, err
	}
	prices := make(map[time.Time]*model.Price)
	for _, d := range f.Directives {
		if p, ok := d.Directive.(syntax.Price); ok {
			m, err := price.Create(ctx, &p)
			if err != nil {
				return nil, err
			}
			prices[m.Date] = m
		} else {
			return nil, fmt.Errorf("unexpected directive in prices file: %v", d)
		}
	}
	return prices, nil
}

func (r *fetchRunner) fetchPrices(reg *registry.Registry, cfg fetchConfig, t0, t1 time.Time, results map[time.Time]*model.Price) error {
	var (
		c                 = yahoo.New()
		quotes            []yahoo.Quote
		commodity, target *model.Commodity
		err               error
	)
	if quotes, err = c.Fetch(cfg.Symbol, t0, t1); err != nil {
		return err
	}
	if commodity, err = reg.Commodities().Get(cfg.Commodity); err != nil {
		return err
	}
	if target, err = reg.Commodities().Get(cfg.TargetCommodity); err != nil {
		return err
	}
	for _, quote := range quotes {
		results[quote.Date] = &model.Price{
			Date:      quote.Date,
			Commodity: commodity,
			Target:    target,
			Price:     decimal.NewFromFloat(quote.Close),
		}
	}
	return nil
}

func (r *fetchRunner) writeFile(ctx *registry.Registry, prices map[time.Time]*model.Price, filepath string) error {
	j := journal.New(ctx)
	for _, price := range prices {
		j.Add(price)
	}
	var buf bytes.Buffer
	err := journal.Print(&buf, j)
	if err != nil {
		return err
	}
	return atomic.WriteFile(filepath, &buf)
}

type fetchConfig struct {
	Symbol          string `yaml:"symbol"`
	File            string `yaml:"file"`
	Commodity       string `yaml:"commodity"`
	TargetCommodity string `yaml:"target_commodity"`
}
