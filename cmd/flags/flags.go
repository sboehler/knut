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

package flags

import (
	"fmt"
	"regexp"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/multierr"

	"github.com/sboehler/knut/lib/date"
	"github.com/sboehler/knut/lib/ledger"
)

// GetAccountFlag is a helper to get an account passed as a flag to the command.
func GetAccountFlag(cmd *cobra.Command, as ledger.Context, flag string) (*ledger.Account, error) {
	name, err := cmd.Flags().GetString(flag)
	if err != nil {
		return nil, err
	}
	return as.GetAccount(name)
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
func GetCommodityFlag(cmd *cobra.Command, ctx ledger.Context, name string) (*ledger.Commodity, error) {
	s, err := cmd.Flags().GetString(name)
	if err != nil {
		return nil, err
	}
	return ctx.GetCommodity(s)
}

// GetPeriodFlag parses a period from a set of flags.
func GetPeriodFlag(cmd *cobra.Command) (date.Period, error) {
	var (
		periods = []struct {
			name   string
			period date.Period
		}{
			{"days", date.Daily},
			{"weeks", date.Weekly},
			{"months", date.Monthly},
			{"quarters", date.Quarterly},
			{"years", date.Yearly},
		}

		errors  error
		results []date.Period
	)
	for _, tuple := range periods {
		v, err := cmd.Flags().GetBool(tuple.name)
		if err != nil {
			errors = multierr.Append(errors, err)
			continue
		}
		if v {
			results = append(results, tuple.period)
		}
	}
	if errors != nil {
		return date.Once, errors
	}
	if len(results) > 1 {
		return date.Once, fmt.Errorf("received multiple conflicting periods: %v", results)
	} else if len(results) == 0 {
		return date.Once, nil
	}
	return results[0], nil
}
