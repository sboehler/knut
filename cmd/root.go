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

// Package cmd is the main command file for Cobra
package cmd

import (
	"github.com/sboehler/knut/cmd/commands"

	"github.com/spf13/cobra"
)

// CreateCmd creates the command.
func CreateCmd(version string) *cobra.Command {
	c := &cobra.Command{
		Use:     "knut",
		Short:   "knut is a plain text accounting tool",
		Long:    `knut is a plain text accounting tool for tracking personal finances and investments.`,
		Version: version,
	}
	c.AddCommand(commands.CreateBalanceCommand())
	c.AddCommand(commands.CreateCheckCommand())
	c.AddCommand(commands.CreateCompletionCommand(c))
	c.AddCommand(commands.CreateFormatCommand())
	c.AddCommand(commands.CreateImportCommand())
	c.AddCommand(commands.CreateInferCmd())
	c.AddCommand(commands.CreatePortfolioCommand())
	c.AddCommand(commands.CreateFetchCommand())
	c.AddCommand(commands.CreateRegisterCmd())
	c.AddCommand(commands.CreateTranscodeCommand())
	c.AddCommand(commands.CreatePrintCommand())

	return c
}
