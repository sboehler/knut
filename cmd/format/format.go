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

package format

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/natefinch/atomic"
	"github.com/spf13/cobra"
	"go.uber.org/multierr"

	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
	"github.com/sboehler/knut/lib/journal/ast/format"
	"github.com/sboehler/knut/lib/journal/ast/parser"
)

// CreateCmd creates the command.
func CreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "format",
		Short: "Format the given journal",
		Long:  `Format the given journal in-place. Any white space and comments between directives is preserved.`,

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
			if cpr.Push(ctx, sema, true) != nil {
			}
			go func(arg string) {
				if err := formatFile(ctx, arg); err != nil {
					if cpr.Push(ctx, errCh, err) != nil {
						return
					}
				}
				if _, _, err := cpr.Pop(ctx, sema); err != nil {
					return
				}
			}(arg)
		}
		for i := 0; i < concurrency; i++ {
			if cpr.Push(ctx, sema, true) != nil {
				return
			}
		}
	}()

	var errors error
	for err := range errCh {
		errors = multierr.Append(errors, err)
	}
	return errors
}

func formatFile(ctx context.Context, target string) error {
	var (
		directives           []ast.Directive
		err                  error
		srcFile, tmpDestFile *os.File
	)
	if directives, err = readDirectives(ctx, target); err != nil {
		return err
	}
	if srcFile, err = os.Open(target); err != nil {
		return err
	}
	if tmpDestFile, err = ioutil.TempFile(path.Dir(target), "format-"); err != nil {
		return multierr.Append(err, srcFile.Close())
	}
	var dest = bufio.NewWriter(tmpDestFile)
	err = format.Format(directives, bufio.NewReader(srcFile), dest)
	err = multierr.Combine(err, srcFile.Close(), dest.Flush(), tmpDestFile.Close())
	if err != nil {
		return multierr.Append(err, os.Remove(tmpDestFile.Name()))
	}
	return multierr.Append(err, atomic.ReplaceFile(tmpDestFile.Name(), target))
}

func readDirectives(ctx context.Context, target string) ([]ast.Directive, error) {
	p, close, err := parser.FromPath(journal.NewContext(), target)
	if err != nil {
		return nil, err
	}
	defer close()

	resCh, errCh := p.Parse(ctx)

	var directives []ast.Directive

	for {
		d, ok, err := cpr.Get(resCh, errCh)
		if !ok {
			break
		}
		if err != nil {
			return nil, err
		}
		directives = append(directives, d)
	}
	return directives, nil
}
