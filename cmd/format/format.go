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
	"bytes"
	"fmt"
	"os"

	"github.com/natefinch/atomic"
	"github.com/spf13/cobra"
	"go.uber.org/multierr"

	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/syntax"
	"github.com/sboehler/knut/lib/syntax/parser"
	"github.com/sboehler/knut/lib/syntax/printer"
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
	file, err := readDirectives(target)
	if err != nil {
		return err
	}
	var (
		dest bytes.Buffer
		p    printer.Printer
	)
	if err := p.Format(file, &dest); err != nil {
		return err
	}
	return atomic.WriteFile(target, &dest)
}

func readDirectives(target string) (syntax.File, error) {
	text, err := os.ReadFile(target)
	if err != nil {
		return syntax.File{}, err
	}
	p := parser.New(string(text), target)
	if err := p.Advance(); err != nil {
		return syntax.File{}, err
	}
	return p.ParseFile()
}
