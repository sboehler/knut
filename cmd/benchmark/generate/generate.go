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

package generate

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	"github.com/sboehler/knut/cmd/flags"
	"github.com/sboehler/knut/lib/common/date"
	"github.com/sboehler/knut/lib/journal"
	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"
)

// CreateCmd creates the command.
func CreateCmd() *cobra.Command {
	var c config
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate a benchmark journal",
		Long:  `Generate a benchmark journal.`,
		Args:  cobra.ExactValidArgs(1),
		Run:   c.run,
	}
	c.setupFlags(cmd)
	return cmd
}

func (c *config) run(cmd *cobra.Command, args []string) {
	if err := c.execute(cmd, args); err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), err)
		os.Exit(1)
	}
}

type config struct {
	context                                       journal.Context
	accounts, transactions, includes, commodities int
	seed                                          int64
	period                                        flags.PeriodFlag
	path                                          string
}

func (c *config) setupFlags(cmd *cobra.Command) {
	c.period.Setup(cmd, date.Period{Start: date.Date(2010, 1, 1), End: date.Date(2010, 12, 31)})

	cmd.Flags().IntVar(&c.accounts, "accounts", 100, "number of accounts to generate")
	cmd.Flags().IntVar(&c.commodities, "commodities", 10, "number of commodities to generate")
	cmd.Flags().IntVar(&c.transactions, "transactions", 1000000, "number of transactions to generate")
	cmd.Flags().Int64Var(&c.seed, "seed", 1, "random seed")
	cmd.Flags().IntVar(&c.includes, "includes", 10, "number of include files (use 0 to disable)")
}

func (c *config) execute(cmd *cobra.Command, args []string) error {
	if c.includes < 0 {
		return fmt.Errorf("includes must be nonnegative")
	}
	open, price, trx := c.generate()
	if err := os.Mkdir(c.path, 0755); err != nil {
		return err
	}
	var files []io.Writer
	j, close, err := createFile(filepath.Join(c.path, "journal.knut"))
	if err != nil {
		return err
	}
	defer close()
	defer j.Flush()

	var p journal.Printer

	if c.includes == 0 {
		files = append(files, j)
	} else {
		for i := 0; i < c.includes; i++ {
			name := fmt.Sprintf("include%d.knut", i)
			include, close, err := createFile(filepath.Join(c.path, name))
			if err != nil {
				return err
			}
			defer close()
			defer include.Flush()
			files = append(files, include)
			if _, err := p.PrintDirective(j, &journal.Include{Path: name}); err != nil {
				return err
			}
			io.WriteString(j, "\n")
		}
	}
	for i, o := range open {
		if _, err := p.PrintDirective(files[i%len(files)], o); err != nil {
			return err
		}
		io.WriteString(files[i%len(files)], "\n")
	}
	for i, o := range price {
		if _, err := p.PrintDirective(files[i%len(files)], o); err != nil {
			return err
		}
		io.WriteString(files[i%len(files)], "\n")
	}
	for i, o := range trx {
		if _, err := p.PrintDirective(files[i%len(files)], o); err != nil {
			return err
		}
		io.WriteString(files[i%len(files)], "\n")
	}
	return nil
}

func createFile(path string) (*bufio.Writer, func() error, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, nil, err
	}
	return bufio.NewWriter(f), func() error { return f.Close() }, nil

}

func (c *config) generate() ([]*journal.Open, []*journal.Price, []*journal.Transaction) {
	rand.Seed(c.seed)
	var (
		accounts     = c.generateAccounts()
		commodities  = c.generateCommodities()
		opens        = c.generateOpenings(accounts)
		prices       = c.generatePrices(commodities)
		transactions = c.generateTransactions(commodities, accounts)
	)
	return opens, prices, transactions
}

func (c *config) generateAccounts() []*journal.Account {
	var (
		as    []*journal.Account
		types = []string{"Assets", "Liabilities", "Income", "Expenses"}
	)
	for i := 0; i < c.accounts; i++ {
		var s strings.Builder
		s.WriteString(types[rand.Intn(4)])
		s.WriteRune(':')
		s.WriteString(c.generateIdentifier(10))
		a, err := c.context.GetAccount(s.String())
		if err != nil {
			panic(fmt.Sprintf("Could not create account %s", s.String()))
		}
		as = append(as, a)
	}
	return as
}

func (c *config) generateCommodities() []*journal.Commodity {
	var res []*journal.Commodity
	for i := 0; i < c.commodities; i++ {
		commodity, err := c.context.GetCommodity(fmt.Sprintf("COMMODITY%d", i))
		if err != nil {
			panic("invalid commodity")
		}
		res = append(res, commodity)
	}
	return res
}

func (c *config) generateOpenings(as []*journal.Account) []*journal.Open {
	var res []*journal.Open

	for _, a := range as {
		res = append(res, &journal.Open{
			Date:    c.period.Value().Start,
			Account: a,
		})
	}
	return res
}

func (c *config) generateTransactions(cs []*journal.Commodity, as []*journal.Account) []*journal.Transaction {
	var trx []*journal.Transaction
	dates := c.period.Value().Dates(date.Daily, 0)
	for i := 0; i < c.transactions; i++ {
		trx = append(trx, journal.TransactionBuilder{
			Date:        dates[rand.Intn(len(dates))],
			Description: c.generateIdentifier(200),
			Postings: journal.PostingBuilder{
				Credit:    as[rand.Intn(len(as))],
				Debit:     as[rand.Intn(len(as))],
				Commodity: cs[rand.Intn(len(cs))],
				Amount:    decimal.NewFromFloat(rand.Float64() * 1000).Round(4),
			}.Build(),
		}.Build())
	}
	return trx
}

var stdev = 0.13 / math.Sqrt(365)

func (c *config) generatePrices(cs []*journal.Commodity) []*journal.Price {
	var prices []*journal.Price
	for _, commodity := range cs[1:] {
		price := decimal.NewFromFloat(1.0 + 200*rand.Float64())
		for _, d := range c.period.Value().Dates(date.Daily, 0) {
			price = price.Mul(decimal.NewFromFloat(1 + rand.NormFloat64()*stdev)).Truncate(4)
			prices = append(prices, &journal.Price{
				Date:      d,
				Commodity: commodity,
				Target:    cs[0],
				Price:     price,
			})
		}
	}
	return prices
}

var small = []rune("abcdefghijklmnopqrstuvwxyz")
var large []rune = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")

func (c *config) generateIdentifier(n int) string {
	var s strings.Builder
	s.WriteRune(large[rand.Intn(len(large))])
	for i := 0; i < n-1; i++ {
		s.WriteRune(small[rand.Intn(len(large))])
	}
	return s.String()
}
