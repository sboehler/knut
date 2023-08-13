package importer

import "github.com/spf13/cobra"

var importers []func() *cobra.Command

// RegisterImporter registers an importer constructor.
func RegisterImporter(f func() *cobra.Command) {
	importers = append(importers, f)
}

func GetImporters() []func() *cobra.Command {
	return importers
}
