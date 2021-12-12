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

package infer

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"

	"github.com/natefinch/atomic"
	"github.com/spf13/cobra"
	"go.uber.org/multierr"

	"github.com/sboehler/knut/cmd/flags"
	"github.com/sboehler/knut/lib/bayes"
	"github.com/sboehler/knut/lib/format"
	"github.com/sboehler/knut/lib/ledger"
	"github.com/sboehler/knut/lib/parser"
)

// CreateCmd creates the command.
func CreateCmd() *cobra.Command {
	var r runner
	var cmd = &cobra.Command{
		Use:   "infer",
		Short: "Auto-assign accounts in a journal",
		Long: `Build a Bayes model using the supplied training file and apply it to replace
		the indicated account in the target file. Training file and target file may be the same.`,
		Args: cobra.ExactValidArgs(1),
		Run:  r.run,
	}
	r.setupFlags(cmd)
	return cmd
}

type runner struct {
	account      flags.AccountFlag
	trainingFile string
	inplace      bool
}

func (r *runner) setupFlags(cmd *cobra.Command) {
	cmd.Flags().VarP(&r.account, "account", "a", "account name")
	cmd.Flags().BoolVarP(&r.inplace, "inplace", "i", false, "infer the accounts inplace")
	cmd.Flags().StringVarP(&r.trainingFile, "training-file", "t", "", "the journal file with existing data")
	cmd.MarkFlagRequired("training-file")
}

func (r *runner) run(cmd *cobra.Command, args []string) {
	if err := r.execute(cmd, args); err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), err)
		os.Exit(1)
	}
}

func (r *runner) execute(cmd *cobra.Command, args []string) (errors error) {
	var (
		ctx        = ledger.NewContext()
		targetFile = args[0]
		account    *ledger.Account
		err        error
	)
	tbd, _ := ctx.GetAccount("Expenses:TBD")
	if account, err = r.account.ValueWithDefault(ctx, tbd); err != nil {
		return err
	}
	model, err := train(ctx, r.trainingFile, account)
	if err != nil {
		return err
	}
	directives, err := r.parseAndInfer(ctx, model, targetFile, account)
	if err != nil {
		return err
	}
	if r.inplace {
		tmpFile, err := r.writeToTmp(directives, targetFile)
		if err != nil {
			return err
		}
		return atomic.ReplaceFile(tmpFile, targetFile)
	}
	out := bufio.NewWriter(cmd.OutOrStdout())
	defer out.Flush()
	return r.writeTo(directives, targetFile, out)
}

func train(ctx ledger.Context, file string, exclude *ledger.Account) (*bayes.Model, error) {
	var (
		j = parser.RecursiveParser{Context: ctx, File: file}
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

func (r *runner) parseAndInfer(ctx ledger.Context, model *bayes.Model, targetFile string, account *ledger.Account) ([]ledger.Directive, error) {
	p, cls, err := parser.FromPath(ctx, targetFile)
	if err != nil {
		return nil, err
	}
	defer cls()
	var directives []ledger.Directive
	for i := range p.ParseAll() {
		switch d := i.(type) {
		case *ledger.Transaction:
			model.Infer(d, account)
			directives = append(directives, d)
		case ledger.Directive:
			directives = append(directives, d)
		default:
			return nil, multierr.Append(cls(), fmt.Errorf("unknown directive: %s", d))
		}
	}
	return directives, nil
}

func (r *runner) writeToTmp(directives []ledger.Directive, targetFile string) (string, error) {
	tmpfile, err := ioutil.TempFile(path.Dir(targetFile), "infer-")
	if err != nil {
		return "", err
	}
	defer tmpfile.Close()

	var dest = bufio.NewWriter(tmpfile)
	defer dest.Flush()

	return tmpfile.Name(), r.writeTo(directives, targetFile, dest)
}

func (r *runner) writeTo(directives []ledger.Directive, targetFile string, out io.Writer) error {
	srcFile, err := os.Open(targetFile)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	return format.Format(directives, bufio.NewReader(srcFile), out)
}
