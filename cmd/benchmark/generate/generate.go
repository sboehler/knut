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
	"time"

	"github.com/sboehler/knut/lib/common/date"
	"github.com/sboehler/knut/lib/journal"
	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"
)

// CreateCmd creates the command.
func CreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate a benchmark journal",
		Long:  `Generate a benchmark journal.`,
		Args:  cobra.ExactValidArgs(1),
		Run:   run,
	}
	cmd.Flags().Int("accounts", 100, "number of accounts to generate")
	cmd.Flags().Int("commodities", 10, "number of commodities to generate")
	cmd.Flags().Int("transactions", 1000000, "number of transactions to generate")
	cmd.Flags().Int64("seed", 1, "random seed")
	cmd.Flags().String("from", "2010-01-01", "from date")
	cmd.Flags().String("to", "2020-12-31", "to date")
	cmd.Flags().Int("includes", 10, "number of include files (use 0 to disable)")

	return cmd
}

func run(cmd *cobra.Command, args []string) {
	if err := execute(cmd, args); err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), err)
		os.Exit(1)
	}
}

type config struct {
	context                                       journal.Context
	accounts, transactions, includes, commodities int
	seed                                          int64
	from, to                                      time.Time
	path                                          string
}

func execute(cmd *cobra.Command, args []string) error {
	c, err := readConfig(cmd, args)
	if err != nil {
		return err
	}
	open, price, trx := generate(c)
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
			var name = fmt.Sprintf("include%d.knut", i)
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

func readConfig(cmd *cobra.Command, args []string) (config, error) {
	var (
		c   config
		err error
	)
	c.context = journal.NewContext()
	if c.accounts, err = cmd.Flags().GetInt("accounts"); err != nil {
		return c, err
	}
	if c.commodities, err = cmd.Flags().GetInt("commodities"); err != nil {
		return c, err
	}
	if c.transactions, err = cmd.Flags().GetInt("transactions"); err != nil {
		return c, err
	}
	if c.includes, err = cmd.Flags().GetInt("includes"); err != nil {
		return c, err
	}
	if c.includes < 0 {
		return c, fmt.Errorf("includes must be nonnegative")
	}
	if c.seed, err = cmd.Flags().GetInt64("seed"); err != nil {
		return c, err
	}
	if c.from, err = parseDate(cmd, "from"); err != nil {
		return c, err
	}
	if c.to, err = parseDate(cmd, "to"); err != nil {
		return c, err
	}
	c.path = args[0]
	return c, nil
}

func parseDate(cmd *cobra.Command, name string) (time.Time, error) {
	s, err := cmd.Flags().GetString(name)
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse("2006-01-02", s)
}

func generate(c config) ([]*journal.Open, []*journal.Price, []*journal.Transaction) {
	rand.Seed(c.seed)
	var (
		accounts     = generateAccounts(c)
		commodities  = generateCommodities(c)
		opens        = generateOpenings(c, accounts)
		prices       = generatePrices(c, commodities)
		transactions = generateTransactions(c, commodities, accounts)
	)
	return opens, prices, transactions
}

func generateAccounts(c config) []*journal.Account {
	var (
		as    []*journal.Account
		types = []string{"Assets", "Liabilities", "Income", "Expenses"}
	)
	for i := 0; i < c.accounts; i++ {
		var s strings.Builder
		s.WriteString(types[rand.Intn(4)])
		s.WriteRune(':')
		s.WriteString(generateIdentifier(10))
		a, err := c.context.GetAccount(s.String())
		if err != nil {
			panic(fmt.Sprintf("Could not create account %s", s.String()))
		}
		as = append(as, a)
	}
	return as
}

func generateCommodities(c config) []*journal.Commodity {
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

func generateOpenings(c config, as []*journal.Account) []*journal.Open {
	var res []*journal.Open

	for _, a := range as {
		res = append(res, &journal.Open{
			Date:    c.from,
			Account: a,
		})
	}
	return res
}

func generateTransactions(c config, cs []*journal.Commodity, as []*journal.Account) []*journal.Transaction {
	var trx []*journal.Transaction
	dates := date.CreatePartition(c.from, c.to, date.Daily, 0).EndDates()
	for i := 0; i < c.transactions; i++ {
		trx = append(trx, journal.TransactionBuilder{
			Date:        dates[rand.Intn(len(dates))],
			Description: generateIdentifier(200),
			Postings: journal.PostingBuilder{
				Credit:    as[rand.Intn(len(as))],
				Debit:     as[rand.Intn(len(as))],
				Commodity: cs[rand.Intn(len(cs))],
				Amount:    decimal.NewFromFloat(rand.Float64() * 1000).Round(4),
			}.Singleton(),
		}.Build())
	}
	return trx
}

var stdev = 0.13 / math.Sqrt(365)

func generatePrices(c config, cs []*journal.Commodity) []*journal.Price {
	var prices []*journal.Price
	for _, commodity := range cs[1:] {
		price := decimal.NewFromFloat(1.0 + 200*rand.Float64())
		for _, d := range date.CreatePartition(c.from, c.to, date.Daily, 0).EndDates() {
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

func generateIdentifier(n int) string {
	var s strings.Builder
	s.WriteRune(large[rand.Intn(len(large))])
	for i := 0; i < n-1; i++ {
		s.WriteRune(small[rand.Intn(len(large))])
	}
	return s.String()
}
