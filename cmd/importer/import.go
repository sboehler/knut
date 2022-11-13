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

package importer

import (
	"github.com/spf13/cobra"
)

// CreateCmd is the import command.
func CreateCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "import",
		Short: "Import financial account statements",
	}
	for _, constructor := range importers {
		cmd.AddCommand(constructor())
	}
	return &cmd
}

var importers []func() *cobra.Command

// Register registers an importer constructor.
func Register(f func() *cobra.Command) {
	importers = append(importers, f)
}
