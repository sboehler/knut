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

package main

import (
	"io/ioutil"
	"os"
	"strings"
	"text/template"

	"github.com/sboehler/knut/cmd"

	// enable importers here
	_ "github.com/sboehler/knut/cmd/importer/cumulus"
	_ "github.com/sboehler/knut/cmd/importer/interactivebrokers"
	_ "github.com/sboehler/knut/cmd/importer/postfinance"
	_ "github.com/sboehler/knut/cmd/importer/revolut"
	_ "github.com/sboehler/knut/cmd/importer/swisscard"
	_ "github.com/sboehler/knut/cmd/importer/swissquote"
)

type config struct {
	ExampleFile string
	PricesFile  string
	Commands    map[string]string
}

type command struct {
	args []string
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
	c := &config{
		Commands: make(map[string]string),
	}
	content, err := ioutil.ReadFile("doc/example.knut")
	if err != nil {
		return nil, err
	}
	c.ExampleFile = string(content)
	content, err = ioutil.ReadFile("doc/prices.yaml")
	if err != nil {
		return nil, err
	}
	c.PricesFile = string(content)

	c.Commands["help"] = run([]string{"--help"})
	c.Commands["HelpImport"] = run([]string{"import", "--help"})

	c.Commands["BalanceIntro"] = run([]string{"balance",
		"-v", "CHF", `-c0,(Income|Expenses|Equity)`, "--monthly", "--from",
		"2020-01-01", "--to", "2020-04-01", "doc/example.knut",
	})
	c.Commands["BalanceMonthlyCHF"] = run([]string{"balance",
		"-v", "CHF", "--monthly", "--to", "2020-04-01", "doc/example.knut",
	})
	c.Commands["IncomeMonthlyCHF"] = run([]string{"balance",
		"-v", "CHF", "--monthly", "--to", "2020-04-01", "-c0,(Assets|Liabilities)", "--diff", "doc/example.knut",
	})
	c.Commands["BalanceBasic"] = run([]string{"balance", "doc/example.knut"})
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
	c := cmd.CreateCmd()
	c.SetArgs(args)
	b := strings.Builder{}
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
