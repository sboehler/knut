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
	"fmt"
	"os"

	"github.com/sboehler/knut/cmd/balance"
	"github.com/sboehler/knut/cmd/format"
	"github.com/sboehler/knut/cmd/importer"
	"github.com/sboehler/knut/cmd/infer"
	"github.com/sboehler/knut/cmd/prices"

	"github.com/spf13/cobra"
)

// RootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "knut",
	Short: "knut is a plain text accounting tool",
	Long:  `knut is a plain text accounting tool for tracking personal finances and investments.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprint(rootCmd.ErrOrStderr(), err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(balance.Cmd)
	rootCmd.AddCommand(importer.Cmd)
	rootCmd.AddCommand(prices.Cmd)
	rootCmd.AddCommand(format.Cmd)
	rootCmd.AddCommand(infer.Cmd)
}
