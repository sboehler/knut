package flags

import (
	"regexp"
	"time"

	"github.com/spf13/cobra"

	"github.com/sboehler/knut/lib/ledger"
	"github.com/sboehler/knut/lib/model/accounts"
	"github.com/sboehler/knut/lib/model/commodities"
)

// GetAccountFlag is a helper to get an account passed as a flag to the command.
func GetAccountFlag(cmd *cobra.Command, as ledger.Context, flag string) (*accounts.Account, error) {
	name, err := cmd.Flags().GetString(flag)
	if err != nil {
		return nil, err
	}
	return as.Get(name)
}

// GetDateFlag is a helper to get a date passed as a flag to the command.
func GetDateFlag(cmd *cobra.Command, flag string) (*time.Time, error) {
	s, err := cmd.Flags().GetString(flag)
	if err != nil {
		return nil, err
	}
	t, err := time.Parse("2006-01-02", s)
	return &t, err
}

// GetRegexFlag is a helper to get a regex passed as a flag to the command.
func GetRegexFlag(cmd *cobra.Command, flag string) (*regexp.Regexp, error) {
	s, err := cmd.Flags().GetString(flag)
	if err != nil {
		return nil, err
	}
	return regexp.Compile(s)
}

// GetCommodityFlag is a helper to get a commodity passed as a flag to the command.
func GetCommodityFlag(cmd *cobra.Command, name string) (*commodities.Commodity, error) {
	s, err := cmd.Flags().GetString(name)
	if err != nil {
		return nil, err
	}
	return commodities.Get(s)
}
