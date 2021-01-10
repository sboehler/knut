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
	"io"
	"io/ioutil"
	"os"
	"path"

	"github.com/spf13/cobra"
	"go.uber.org/multierr"

	"github.com/sboehler/knut/lib/bayes"
	"github.com/sboehler/knut/lib/format"
	"github.com/sboehler/knut/lib/ledger"
	"github.com/sboehler/knut/lib/model"
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

		RunE: run,
	}
	cmd.Flags().StringP("account", "a", "TBD", "account name")
	cmd.Flags().StringP("training-file", "t", "", "the journal file with existing data")
	return &cmd
}

type directive interface {
	io.WriterTo
	Position() model.Range
}

func run(cmd *cobra.Command, args []string) (errors error) {
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
	ch := make(chan interface{}, 100)
	go func() {
		defer close(ch)
		upstream, err := parser.ParseOneFile(targetFile)
		if err != nil {
			ch <- err
			return
		}
		for d := range upstream {
			if t, ok := d.(*ledger.Transaction); ok {
				bayesModel.Infer(t, account)
			}
			ch <- d
		}
	}()

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
	err = format.Format(ch, src, dest)
	if err = multierr.Combine(err, dest.Flush(), srcFile.Close()); err != nil {
		return multierr.Append(err, os.Remove(tmpfile.Name()))
	}
	return os.Rename(tmpfile.Name(), targetFile)
}

func train(file string, exclude *accounts.Account) (*bayes.Model, error) {
	ch, err := parser.Parse(file)
	if err != nil {
		return nil, err
	}
	m := bayes.NewModel()
	for r := range ch {
		switch t := r.(type) {
		case error:
			return nil, t
		case *ledger.Transaction:
			m.Update(t)
		}
	}
	return m, nil
}
