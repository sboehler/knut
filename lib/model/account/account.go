package account

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/sboehler/knut/lib/common/compare"
	"github.com/sboehler/knut/lib/common/mapper"
	"github.com/sboehler/knut/lib/common/regex"
)

// Type is the type of an account.
type Type int

const (
	// ASSETS represents an asset account.
	ASSETS Type = iota
	// LIABILITIES represents a liability account.
	LIABILITIES
	// EQUITY represents an equity account.
	EQUITY
	// INCOME represents an income account.
	INCOME
	// EXPENSES represents an expenses account.
	EXPENSES
)

func (t Type) String() string {
	switch t {
	case ASSETS:
		return "Assets"
	case LIABILITIES:
		return "Liabilities"
	case EQUITY:
		return "Equity"
	case INCOME:
		return "Income"
	case EXPENSES:
		return "Expenses"
	}
	return ""
}

// Types is an array with the ordered accont types.
var Types = []Type{ASSETS, LIABILITIES, EQUITY, INCOME, EXPENSES}

var types = map[string]Type{
	"Assets":      ASSETS,
	"Liabilities": LIABILITIES,
	"Equity":      EQUITY,
	"Expenses":    EXPENSES,
	"Income":      INCOME,
}

// Account represents an account which can be used in bookings.
type Account struct {
	accountType Type
	name        string
	segment     string
	level       int
}

// Split returns the account name split into segments.
func (a *Account) Split() []string {
	return strings.Split(a.name, ":")
}

// Name returns the name of this account.
func (a Account) Name() string {
	return a.name
}

// Segment returns the name of this account.
func (a Account) Segment() string {
	return a.segment
}

// Type returns the account type.
func (a Account) Type() Type {
	return a.accountType
}

// IsAL returns whether this account is an asset or liability account.
func (a Account) IsAL() bool {
	return a.accountType == ASSETS || a.accountType == LIABILITIES
}

// IsIE returns whether this account is an income or expense account.
func (a Account) IsIE() bool {
	return a.accountType == EXPENSES || a.accountType == INCOME
}

func (a Account) String() string {
	return a.name
}

func Compare(a1, a2 *Account) compare.Order {
	o := compare.Ordered(a1.accountType, a2.accountType)
	if o != compare.Equal {
		return o
	}
	return compare.Ordered(a1.name, a2.name)
}

// Rule is a rule to shorten accounts which match the given regex.
type Rule struct {
	Level int
	Regex *regexp.Regexp
}

func (r Rule) String() string {
	return fmt.Sprintf("%d,%v", r.Level, r.Regex)
}

// Mapping is a set of mapping rules.
type Mapping []Rule

func (m Mapping) String() string {
	var s []string
	for _, r := range m {
		s = append(s, r.String())
	}
	return strings.Join(s, ", ")
}

// level returns the level to which an account should be shortened.
func (m Mapping) level(a *Account) int {
	for _, c := range m {
		if c.Regex == nil || c.Regex.MatchString(a.name) {
			return c.Level
		}
	}
	return a.level
}

func Shorten(reg *Registry, m Mapping) mapper.Mapper[*Account] {
	if len(m) == 0 {
		return mapper.Identity[*Account]
	}
	return func(a *Account) *Account {
		level := m.level(a)
		if level >= a.level {
			return a
		}
		return reg.NthParent(a, a.level-level)
	}
}

func Remap(reg *Registry, rs regex.Regexes) mapper.Mapper[*Account] {
	return func(a *Account) *Account {
		if rs.MatchString(a.name) {
			return reg.SwapType(a)
		}
		return a
	}
}
