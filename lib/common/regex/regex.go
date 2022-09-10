package regex

import "regexp"

type Regexes []*regexp.Regexp

func (rxs *Regexes) Add(r *regexp.Regexp) {
	*rxs = append(*rxs, r)
}

// Value returns the flag value.
func (rf Regexes) MatchString(s string) bool {
	for _, r := range rf {
		if r.MatchString(s) {
			return true
		}
	}
	return false
}
