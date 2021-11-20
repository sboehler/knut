package ledger

import (
	"regexp"
)

// Filter represents a filter creating a
type Filter struct {
	AccountsFilter, CommoditiesFilter *regexp.Regexp
}

// MatchAccount returns whether this filterthe given Account.
func (b Filter) MatchAccount(a *Account) bool {
	return b.AccountsFilter == nil || b.AccountsFilter.MatchString(a.String())
}

// MatchCommodity returns whether this filter matches the given Commodity.
func (b Filter) MatchCommodity(c *Commodity) bool {
	return b.CommoditiesFilter == nil || b.CommoditiesFilter.MatchString(c.String())
}
