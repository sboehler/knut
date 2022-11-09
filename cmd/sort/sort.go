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

package sort

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/natefinch/atomic"
	"github.com/spf13/cobra"
	"go.uber.org/multierr"

	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/parser"
)

// CreateCmd creates the command.
func CreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sort",
		Short: "sort the given files",
		Long:  `Sort the given journal in-place. No white space and comments between directives is preserved.`,

		Run: run,
	}
}

const concurrency = 10

func run(cmd *cobra.Command, args []string) {
	if err := execute(cmd, args); err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), err)
		os.Exit(1)
	}
}

func execute(cmd *cobra.Command, args []string) error {
	var (
		ctx   = cmd.Context()
		errCh = make(chan error)
	)
	go func() {
		defer close(errCh)
		sema := make(chan bool, concurrency)
		defer close(sema)
		for _, arg := range args {
			sema <- true
			go func(arg string) {
				defer func() { <-sema }()
				if err := sortFile(arg); err != nil {
					if cpr.Push(ctx, errCh, err) != nil {
						return
					}
				}
			}(arg)
		}
		for i := 0; i < concurrency; i++ {
			sema <- true
		}
	}()

	var errors error
	for err := range errCh {
		errors = multierr.Append(errors, err)
	}
	return errors
}

func sortFile(target string) error {
	jctx := journal.NewContext()
	ast, err := readDirectives(jctx, target)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	_, err = journal.NewPrinter().PrintLedger(&buf, ast.SortedDays())
	if err != nil {
		return err
	}
	return atomic.WriteFile(target, &buf)
}

func readDirectives(jctx journal.Context, target string) (*journal.Journal, error) {
	p, close, err := parser.FromPath(jctx, target)
	if err != nil {
		return nil, err
	}
	defer close()

	res := journal.New(jctx)

	for {
		d, err := p.Next()
		if err == io.EOF {
			return res, nil
		}
		if err != nil {
			return nil, err
		}
		switch t := d.(type) {

		case *journal.Open:
			res.AddOpen(t)

		case *journal.Price:
			res.AddPrice(t)

		case *journal.Transaction:
			res.AddTransaction(t)

		case *journal.Assertion:
			res.AddAssertion(t)

		case *journal.Value:
			res.AddValue(t)

		case *journal.Close:
			res.AddClose(t)

		default:
			return nil, fmt.Errorf("unknown: %#v", t)
		}
	}
}
