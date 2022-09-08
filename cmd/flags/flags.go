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
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/pflag"

	"github.com/sboehler/knut/lib/common/date"
	"github.com/sboehler/knut/lib/journal"
)

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

func (tf DateFlag) ValueOr(t time.Time) time.Time {
	v := tf.Value()
	if v.IsZero() {
		return t
	}
	return v
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

// IntervalFlags manages multiple flags to determine a time period.
type IntervalFlags struct {
	flags [6]bool
}

// Setup configures the flags.
func (pf *IntervalFlags) Setup(s *pflag.FlagSet) {
	s.BoolVar(&pf.flags[date.Daily], "days", false, "days")
	s.BoolVar(&pf.flags[date.Weekly], "weeks", false, "weeks")
	s.BoolVar(&pf.flags[date.Monthly], "months", false, "months")
	s.BoolVar(&pf.flags[date.Quarterly], "quarters", false, "quarters")
	s.BoolVar(&pf.flags[date.Yearly], "years", false, "years")
}

// Value returns the period.
func (pf IntervalFlags) Value() (date.Interval, error) {
	var index, counter int
	for i, val := range pf.flags {
		if val {
			counter++
			index = i
		}
	}
	if counter > 1 {
		return date.Once, fmt.Errorf("multiple incompatible intervals")
	}
	return (date.Interval)(index), nil
}

// MappingFlag manages a flag of type -c1,<regex>.
type MappingFlag struct {
	m journal.AccountMapping
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
	cf.m = append(cf.m, journal.Rule{Level: l, Regex: regex})
	return nil
}

// Value returns the value of this flag.
func (cf *MappingFlag) Value() journal.AccountMapping {
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
func (cf CommodityFlag) Value(ctx journal.Context) (*journal.Commodity, error) {
	if cf.val != "" {
		return ctx.GetCommodity(cf.val)
	}
	return nil, nil
}

// AccountFlag manages a flag to parse a commodity.
type AccountFlag struct {
	val string
}

// Set implements pflag.Value.
func (cf *AccountFlag) Set(v string) error {
	cf.val = v
	return nil
}

// Type implements pflag.Value.
func (cf AccountFlag) Type() string {
	return "<account>"
}

// Value returns the flag value.
func (cf AccountFlag) String() string {
	return cf.val
}

// Value returns the account.
func (cf AccountFlag) Value(ctx journal.Context) (*journal.Account, error) {
	if cf.val != "" {
		return ctx.GetAccount(cf.val)
	}
	return nil, nil
}

// ValueWithDefault returns the account. If no account has been specified, the default is returned.
func (cf AccountFlag) ValueWithDefault(ctx journal.Context, def *journal.Account) (*journal.Account, error) {
	res, err := cf.Value(ctx)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return def, nil
	}
	return res, nil
}

// OpenFile opens the file at the given path as a buffered reader.
func OpenFile(p string) (*bufio.Reader, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	return bufio.NewReader(f), nil

}
