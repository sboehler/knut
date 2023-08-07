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
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/natefinch/atomic"
	"github.com/sourcegraph/conc/pool"
	"github.com/spf13/cobra"

	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/syntax"
	"github.com/sboehler/knut/lib/syntax/bayes"
)

// CreateCmd creates the command.
func CreateCmd() *cobra.Command {
	var r runner
	cmd := &cobra.Command{
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
	account      string
	trainingFile string
	inplace      bool
}

func (r *runner) setupFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&r.account, "account", "a", "Expenses:TBD", "account name")
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
		targetFile = args[0]
		err        error
	)
	model, err := train(cmd.Context(), r.trainingFile, r.account)
	if err != nil {
		return err
	}
	file, err := r.parseAndInfer(cmd.Context(), model, targetFile)
	if err != nil {
		return err
	}
	if r.inplace {
		var buf bytes.Buffer
		if err := syntax.FormatFile(&buf, file); err != nil {
			return err
		}
		return atomic.WriteFile(targetFile, &buf)
	} else {
		out := bufio.NewWriter(cmd.OutOrStdout())
		defer out.Flush()
		return syntax.FormatFile(out, file)
	}
}

func train(ctx context.Context, file string, account string) (*bayes.Model, error) {
	model := bayes.NewModel(account)
	p := pool.New().WithErrors().WithFirstError().WithContext(ctx)
	ch, worker := syntax.ParseFileRecursively(file)
	p.Go(worker)
	p.Go(func(ctx context.Context) error {
		return cpr.Consume(ctx, ch, func(res syntax.File) error {
			for _, d := range res.Directives {
				if t, ok := d.Directive.(syntax.Transaction); ok {
					model.Update(&t)
				}
			}
			return nil
		})
	})
	return model, p.Wait()
}

func (r *runner) parseAndInfer(ctx context.Context, model *bayes.Model, targetFile string) (syntax.File, error) {
	f, err := syntax.ParseFile(targetFile)
	if err != nil {
		return syntax.File{}, err
	}
	for i := range f.Directives {
		if t, ok := f.Directives[i].Directive.(syntax.Transaction); ok {
			model.Infer(&t)
		}
	}
	return f, nil
}
