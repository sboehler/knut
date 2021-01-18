package benchmark

import (
	"github.com/spf13/cobra"

	"github.com/sboehler/knut/cmd/benchmark/generate"
)

// CreateCmd is the import command.
func CreateCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "benchmark",
		Short: "various subcommands to benchmark knut",
	}
	cmd.AddCommand(generate.CreateCmd())
	return &cmd
}
