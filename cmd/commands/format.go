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

package commands

import (
	"bytes"
	"fmt"
	"os"

	"github.com/natefinch/atomic"
	"github.com/sourcegraph/conc/iter"
	"github.com/spf13/cobra"
	"go.uber.org/multierr"

	"github.com/sboehler/knut/lib/syntax"
)

// CreateFormatCommand creates the command.
func CreateFormatCommand() *cobra.Command {
	var runner formatRunner
	return &cobra.Command{
		Use:   "format",
		Short: "Format the given journal",
		Long:  `Format the given journal in-place. Any white space and comments between directives is preserved.`,

		Run: runner.run,
	}
}

type formatRunner struct{}

func (r formatRunner) run(cmd *cobra.Command, args []string) {
	if err := r.execute(cmd, args); err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), err)
		os.Exit(1)
	}
}

func (r formatRunner) execute(cmd *cobra.Command, args []string) error {
	return multierr.Combine(iter.Map(args, r.formatFile)...)
}

func (formatRunner) formatFile(target *string) error {
	file, err := syntax.ParseFile(*target)
	if err != nil {
		return err
	}
	var dest bytes.Buffer
	if err := syntax.FormatFile(&dest, file); err != nil {
		return err
	}
	return atomic.WriteFile(*target, &dest)
}
