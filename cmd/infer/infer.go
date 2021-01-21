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

package infer

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/spf13/cobra"
	"go.uber.org/multierr"

	"github.com/sboehler/knut/lib/bayes"
	"github.com/sboehler/knut/lib/format"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/ledger"
	"github.com/sboehler/knut/lib/model/accounts"
	"github.com/sboehler/knut/lib/parser"
)

// CreateCmd creates the command.
func CreateCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "infer",
		Short: "Auto-assign accounts in a journal",
		Long: `Build a Bayes model using the supplied training file and apply it to replace
		the indicated account in the target file. Training file and target file may be the same.`,
		Args: cobra.ExactValidArgs(1),
		Run:  run,
	}
	cmd.Flags().StringP("account", "a", "Expenses:TBD", "account name")
	cmd.Flags().StringP("training-file", "t", "", "the journal file with existing data")
	return &cmd
}

func run(cmd *cobra.Command, args []string) {
	if err := execute(cmd, args); err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), err)
		os.Exit(1)
	}
}

func execute(cmd *cobra.Command, args []string) (errors error) {
	name, err := cmd.Flags().GetString("account")
	if err != nil {
		return err
	}
	account, err := accounts.Get(name)
	if err != nil {
		return err
	}
	trainingFile, err := cmd.Flags().GetString("training-file")
	if err != nil {
		return err
	}
	return infer(trainingFile, args[0], account)
}

func infer(trainingFile string, targetFile string, account *accounts.Account) error {
	bayesModel, err := train(trainingFile, account)
	if err != nil {
		return err
	}
	p, err := parser.Open(targetFile)
	if err != nil {
		return err
	}
	var directives []ledger.Directive
	for i := range p.ParseAll() {
		switch d := i.(type) {
		case *ledger.Transaction:
			bayesModel.Infer(d, account)
			directives = append(directives, d)
		case ledger.Directive:
			directives = append(directives, d)
		default:
			return fmt.Errorf("unknown directive: %s", d)
		}
	}
	srcFile, err := os.Open(targetFile)
	if err != nil {
		return err
	}
	src := bufio.NewReader(srcFile)
	tmpfile, err := ioutil.TempFile(path.Dir(targetFile), "-format")
	if err != nil {
		return err
	}
	dest := bufio.NewWriter(tmpfile)
	err = format.Format(directives, src, dest)
	if err = multierr.Combine(err, dest.Flush(), srcFile.Close()); err != nil {
		return multierr.Append(err, os.Remove(tmpfile.Name()))
	}
	return os.Rename(tmpfile.Name(), targetFile)
}

func train(file string, exclude *accounts.Account) (*bayes.Model, error) {
	var (
		j = journal.Journal{File: file}
		m = bayes.NewModel()
	)
	for r := range j.Parse() {
		switch t := r.(type) {
		case error:
			return nil, t
		case *ledger.Transaction:
			m.Update(t)
		}
	}
	return m, nil
}
