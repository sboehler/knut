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

package web

import (
	"fmt"
	"os"

	"github.com/sboehler/knut/server"
	"github.com/spf13/cobra"
)

// CreateCmd creates the command.
func CreateCmd() *cobra.Command {

	var r runner

	// Cmd is the balance command.
	var c = &cobra.Command{
		Use:    "web",
		Short:  "start the web application",
		Long:   `Start the knut web application.`,
		Run:    r.run,
		Hidden: true,
	}
	r.setupFlags(c)
	return c
}

type runner struct{}

func (r *runner) setupFlags(c *cobra.Command) {
}

func (r *runner) run(cmd *cobra.Command, args []string) {
	if err := server.Test(cmd.OutOrStdout()); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "%+v\n", err)
		os.Exit(1)
	}
}
