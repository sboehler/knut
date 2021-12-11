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
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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
func GetDateFlag(cmd *cobra.Command, flag string) (time.Time, error) {
	s, err := cmd.Flags().GetString(flag)
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse("2006-01-02", s)
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

// GetCollapseFlag parses a flag of type -c1,<regex>.
func GetCollapseFlag(cmd *cobra.Command, name string) (ledger.Mapping, error) {
	collapse, err := cmd.Flags().GetStringArray(name)
	if err != nil {
		return nil, err
	}
	var res = make(ledger.Mapping, 0, len(collapse))
	for _, c := range collapse {
		var s = strings.SplitN(c, ",", 2)
		l, err := strconv.Atoi(s[0])
		if err != nil {
			return nil, fmt.Errorf("expected integer level, got %q (error: %v)", s[0], err)
		}
		var regex *regexp.Regexp
		if len(s) == 2 {
			if regex, err = regexp.Compile(s[1]); err != nil {
				return nil, err
			}
		}
		res = append(res, ledger.Rule{Level: l, Regex: regex})
	}
	return res, nil
}

// DateFlag manages a flag to determine a date.
type DateFlag time.Time

var _ pflag.Value = (*DateFlag)(nil)

func (tf DateFlag) String() string {
	return tf.Value().String()
}

// Set implements pflag.Value.
func (tf *DateFlag) Set(v string) error {
	t, err := time.Parse("2006-01-02", v)
	if err != nil {
		return err
	}
	*tf = (DateFlag)(t)
	return nil
}

// Type implements pflag.Value.
func (tf DateFlag) Type() string {
	return "YYYY-MM-DD"
}

// Value returns the flag value.
func (tf DateFlag) Value() time.Time {
	return time.Time(tf)
}

// RegexFlag manages a flag to get a regex.
type RegexFlag struct {
	regex *regexp.Regexp
}

var _ pflag.Value = (*RegexFlag)(nil)

var _ pflag.Value = (*RegexFlag)(nil)

func (rf RegexFlag) String() string {
	if rf.regex != nil {
		return rf.regex.String()
	}
	return ""
}

// Set implements pflag.Value.
func (rf *RegexFlag) Set(v string) error {
	t, err := regexp.Compile(v)
	if err != nil {
		return err
	}
	rf.regex = t
	return nil
}

// Type implements pflag.Value.
func (rf RegexFlag) Type() string {
	return "<regex>"
}

// Value returns the flag value.
func (rf *RegexFlag) Value() *regexp.Regexp {
	return rf.regex
}

// PeriodFlags manages multiple flags to determine a time period.
type PeriodFlags struct {
	flags [6]bool
}

// Setup configures the flags.
func (pf *PeriodFlags) Setup(s *pflag.FlagSet) {
	s.BoolVar(&pf.flags[date.Daily], "days", false, "days")
	s.BoolVar(&pf.flags[date.Weekly], "weeks", false, "weeks")
	s.BoolVar(&pf.flags[date.Monthly], "months", false, "months")
	s.BoolVar(&pf.flags[date.Quarterly], "quarters", false, "quarters")
	s.BoolVar(&pf.flags[date.Yearly], "years", false, "years")
}

// Value returns the period.
func (pf PeriodFlags) Value() (date.Period, error) {
	var index, counter int
	for i, val := range pf.flags {
		if val {
			counter++
			index = i
		}
	}
	if counter > 1 {
		return date.Once, fmt.Errorf("multiple incompatible time periods")
	}
	return (date.Period)(index), nil
}

// MappingFlag manages a flag of type -c1,<regex>.
type MappingFlag struct {
	m ledger.Mapping
}

var _ pflag.Value = (*MappingFlag)(nil)

func (cf MappingFlag) String() string {
	return cf.m.String()
}

// Type implements pflag.Value.
func (cf MappingFlag) Type() string {
	return "<level>,<regex>"
}

// Set implements pflag.Value.
func (cf *MappingFlag) Set(v string) error {
	var s = strings.SplitN(v, ",", 2)
	l, err := strconv.Atoi(s[0])
	if err != nil {
		return fmt.Errorf("expected integer level, got %q (error: %v)", s[0], err)
	}
	var regex *regexp.Regexp
	if len(s) == 2 {
		if regex, err = regexp.Compile(s[1]); err != nil {
			return err
		}
	}
	cf.m = append(cf.m, ledger.Rule{Level: l, Regex: regex})
	return nil
}

// Value returns the value of this flag.
func (cf *MappingFlag) Value() ledger.Mapping {
	return cf.m
}

// CommodityFlag manages a flag to parse a commodity.
type CommodityFlag struct {
	val string
}

// Set implements pflag.Value.
func (cf *CommodityFlag) Set(v string) error {
	cf.val = v
	return nil
}

// Type implements pflag.Value.
func (cf CommodityFlag) Type() string {
	return "<commodity>"
}

// Value returns the flag value.
func (cf CommodityFlag) String() string {
	return cf.val
}

// Value returns the commodity.
func (cf CommodityFlag) Value(ctx ledger.Context) (*ledger.Commodity, error) {
	if cf.val != "" {
		return ctx.GetCommodity(cf.val)
	}
	return nil, nil
}
