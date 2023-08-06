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

package print

import (
	"bufio"
	"fmt"
	"os"

	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/printer"
	"github.com/sboehler/knut/lib/model/registry"

	"github.com/spf13/cobra"
	"go.uber.org/multierr"
)

// CreateCmd creates the command.
func CreateCmd() *cobra.Command {
	var r runner

	// Cmd is the balance command.
	cmd := &cobra.Command{
		Use:   "print",
		Short: "print the journal",
		Long:  `Print the given journal.`,

		Args: cobra.ExactValidArgs(1),

		Run: r.run,
	}
	r.setupFlags(cmd)
	return cmd
}

type runner struct {
}

func (r *runner) setupFlags(c *cobra.Command) {
}

func (r *runner) run(cmd *cobra.Command, args []string) {
	if err := r.execute(cmd, args); err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), err)
		os.Exit(1)
	}
}

func (r *runner) execute(cmd *cobra.Command, args []string) (errors error) {
	jctx := registry.New()
	j, err := journal.FromPath(cmd.Context(), jctx, args[0])
	if err != nil {
		return err
	}
	_, err = j.Process(
		journal.Balance(jctx, nil),
	)
	w := bufio.NewWriter(cmd.OutOrStdout())
	defer func() { err = multierr.Append(err, w.Flush()) }()

	_, errors = printer.NewPrinter().PrintJournal(w, j)
	return errors
}
