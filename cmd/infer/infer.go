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

	"github.com/natefinch/atomic"
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
	var cmd = cobra.Command{
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
	var (
		s, trainingFile string
		account         *accounts.Account
		err             error
	)
	if s, err = cmd.Flags().GetString("account"); err != nil {
		return err
	}
	if account, err = accounts.Get(s); err != nil {
		return err
	}
	if trainingFile, err = cmd.Flags().GetString("training-file"); err != nil {
		return err
	}
	return infer(trainingFile, args[0], account)
}

func infer(trainingFile string, targetFile string, account *accounts.Account) error {
	bayesModel, err := train(trainingFile, account)
	if err != nil {
		return err
	}
	p, cls, err := parser.FromPath(targetFile)
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
			return multierr.Append(cls(), fmt.Errorf("unknown directive: %s", d))
		}
	}
	if err := cls(); err != nil {
		return err
	}
	srcFile, err := os.Open(targetFile)
	if err != nil {
		return err
	}
	tmpfile, err := ioutil.TempFile(path.Dir(targetFile), "infer-")
	if err != nil {
		return multierr.Append(err, srcFile.Close())
	}
	var dest = bufio.NewWriter(tmpfile)
	err = format.Format(directives, bufio.NewReader(srcFile), dest)
	err = multierr.Combine(err, srcFile.Close(), dest.Flush(), tmpfile.Close())
	if err != nil {
		return multierr.Append(err, os.Remove(tmpfile.Name()))
	}
	return multierr.Append(err, atomic.ReplaceFile(tmpfile.Name(), targetFile))
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
