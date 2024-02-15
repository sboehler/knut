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
	"bufio"
	"fmt"
	"os"

	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/check"
	"github.com/sboehler/knut/lib/model/registry"

	"github.com/spf13/cobra"
)

// CreatePrintCommand creates the command.
func CreatePrintCommand() *cobra.Command {
	var r printRunner

	// Cmd is the balance command.
	cmd := &cobra.Command{
		Use:   "print",
		Short: "print the journal",
		Long:  `Print the given journal.`,

		Args: cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),

		Run: r.run,
	}
	r.setupFlags(cmd)
	return cmd
}

type printRunner struct {
}

func (r *printRunner) setupFlags(c *cobra.Command) {
}

func (r *printRunner) run(cmd *cobra.Command, args []string) {
	if err := r.execute(cmd, args); err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), err)
		os.Exit(1)
	}
}

func (r *printRunner) execute(cmd *cobra.Command, args []string) (errors error) {
	reg := registry.New()
	j, err := journal.FromPath(cmd.Context(), reg, args[0])
	if err != nil {
		return err
	}
	if err := j.Build().Process(check.Check()); err != nil {
		return err
	}
	w := bufio.NewWriter(cmd.OutOrStdout())
	defer w.Flush()
	return journal.Print(w, j.Build())
}
