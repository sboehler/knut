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
	"fmt"
	"io"
	"os"
	"path"

	"github.com/natefinch/atomic"
	"github.com/spf13/cobra"
	"go.uber.org/multierr"

	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/format"
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
			sema <- true
			go func(arg string) {
				defer func() { <-sema }()
				if err := formatFile(arg); err != nil {
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

func formatFile(target string) error {
	var (
		directives           []journal.Directive
		err                  error
		srcFile, tmpDestFile *os.File
	)
	if directives, err = readDirectives(target); err != nil {
		return err
	}
	if srcFile, err = os.Open(target); err != nil {
		return err
	}
	if tmpDestFile, err = os.CreateTemp(path.Dir(target), "format-"); err != nil {
		return multierr.Append(err, srcFile.Close())
	}
	dest := bufio.NewWriter(tmpDestFile)
	err = format.Format(directives, bufio.NewReader(srcFile), dest)
	err = multierr.Combine(err, srcFile.Close(), dest.Flush(), tmpDestFile.Close())
	if err != nil {
		return multierr.Append(err, os.Remove(tmpDestFile.Name()))
	}
	return multierr.Append(err, atomic.ReplaceFile(tmpDestFile.Name(), target))
}

func readDirectives(target string) ([]journal.Directive, error) {
	p, close, err := journal.ParserFromPath(journal.NewContext(), target)
	if err != nil {
		return nil, err
	}
	defer close()

	var directives []journal.Directive

	for {
		d, err := p.Next()
		if err == io.EOF {
			return directives, nil
		}
		if err != nil {
			return nil, err
		}
		directives = append(directives, d)
	}
}
