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

package main

import (
	"os"
	"strings"
	"text/template"

	"github.com/sboehler/knut/cmd"

	// enable importers here
	_ "github.com/sboehler/knut/cmd/importer/cumulus"
	_ "github.com/sboehler/knut/cmd/importer/interactivebrokers"
	_ "github.com/sboehler/knut/cmd/importer/postfinance"
	_ "github.com/sboehler/knut/cmd/importer/revolut"
	_ "github.com/sboehler/knut/cmd/importer/revolut2"
	_ "github.com/sboehler/knut/cmd/importer/supercard"
	_ "github.com/sboehler/knut/cmd/importer/swisscard"
	_ "github.com/sboehler/knut/cmd/importer/swissquote"
	_ "github.com/sboehler/knut/cmd/importer/viac"
)

type config struct {
	ExampleFile string
	PricesFile  string
	Commands    map[string]string
}

func main() {
	c, err := createConfig()
	if err != nil {
		panic(err)
	}
	err = generate(c)
	if err != nil {
		panic(err)
	}
}

func createConfig() (*config, error) {
	var c = &config{
		Commands: make(map[string]string),
	}
	content, err := os.ReadFile("doc/example.knut")
	if err != nil {
		return nil, err
	}
	c.ExampleFile = string(content)
	content, err = os.ReadFile("doc/prices.yaml")
	if err != nil {
		return nil, err
	}
	c.PricesFile = string(content)

	c.Commands["help"] = run([]string{"--help"})
	c.Commands["HelpImport"] = run([]string{"import", "--help"})

	c.Commands["BalanceIntro"] = run([]string{"balance",
		"--color=false", "-v", "CHF", "--months", "--from",
		"2020-01-01", "--to", "2020-04-01", "doc/example.knut",
	})
	c.Commands["FilterAccount"] = run([]string{"balance",
		"--color=false", "-v", "CHF", "--months", "--from",
		"2020-01-01", "--to", "2020-04-01", "--diff", "--account", "Portfolio", "doc/example.knut",
	})
	c.Commands["FilterCommodity"] = run([]string{"balance",
		"--color=false", "-v", "CHF", "--months", "--from",
		"2020-01-01", "--to", "2020-04-01", "--diff", "--commodity", "AAPL", "doc/example.knut",
	})
	c.Commands["Collapse"] = run([]string{"balance",
		"--color=false", "-v", "CHF", "--months", "--from",
		"2020-01-01", "--to", "2020-04-01", "--diff", "-m0,(Income|Expenses)", "doc/example.knut",
	})
	c.Commands["Collapse1"] = run([]string{"balance",
		"--color=false", "-v", "CHF", "--months", "--from",
		"2020-01-01", "--to", "2020-04-01", "--diff", "-m1,(Income|Expenses|Equity)", "doc/example.knut",
	})
	c.Commands["BalanceMonthlyCHF"] = run([]string{"balance",
		"--color=false", "-v", "CHF", "--months", "--to", "2020-04-01", "doc/example.knut",
	})
	c.Commands["BalanceMonthlyUSD"] = run([]string{"balance",
		"--color=false", "-v", "USD", "--months", "--to", "2020-04-01", "doc/example.knut",
	})
	c.Commands["BalanceBasic"] = run([]string{"balance", "--color=false", "doc/example.knut", "--to", "2020-04-01"})
	return c, nil
}

func generate(c *config) error {
	tpl, err := template.ParseFiles("doc/README.md")
	if err != nil {
		return err
	}
	if err = tpl.Execute(os.Stdout, c); err != nil {
		return err
	}
	return nil
}

func run(args []string) string {
	var c = cmd.CreateCmd("development")
	c.SetArgs(args)
	var b strings.Builder
	b.WriteString("$ knut")
	for _, a := range args {
		b.WriteRune(' ')
		b.WriteString(a)
	}
	b.WriteRune('\n')
	c.SetOut(&b)
	//c.SetErr(&b)
	c.Execute()
	return b.String()
}
