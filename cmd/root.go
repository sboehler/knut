// Copyright 2020 Silvio Böhler
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
	"github.com/sboehler/knut/cmd/balance"
	"github.com/sboehler/knut/cmd/benchmark"
	"github.com/sboehler/knut/cmd/completion"
	"github.com/sboehler/knut/cmd/db"
	"github.com/sboehler/knut/cmd/format"
	"github.com/sboehler/knut/cmd/importer"
	"github.com/sboehler/knut/cmd/infer"
	"github.com/sboehler/knut/cmd/prices"
	"github.com/sboehler/knut/cmd/transcode"
	"github.com/sboehler/knut/cmd/web"

	"github.com/spf13/cobra"
)

// CreateCmd creates the command.
func CreateCmd(version string) *cobra.Command {
	var c = &cobra.Command{
		Use:     "knut",
		Short:   "knut is a plain text accounting tool",
		Long:    `knut is a plain text accounting tool for tracking personal finances and investments.`,
		Version: version,
	}
	c.AddCommand(balance.CreateCmd())
	c.AddCommand(importer.CreateCmd())
	c.AddCommand(prices.CreateCmd())
	c.AddCommand(format.CreateCmd())
	c.AddCommand(infer.CreateCmd())
	c.AddCommand(transcode.CreateCmd())
	c.AddCommand(benchmark.CreateCmd())
	c.AddCommand(web.CreateCmd())
	c.AddCommand(completion.CreateCmd(c))
	c.AddCommand(db.CreateCmd())

	return c
}
