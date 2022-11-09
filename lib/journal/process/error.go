package process

import (
	"fmt"
	"strings"

	"github.com/sboehler/knut/lib/journal"
)

// Error is an error.
type Error struct {
	directive journal.Directive
	msg       string
}

func (be Error) Error() string {
	var (
		p journal.Printer
		b strings.Builder
	)
	fmt.Fprintf(&b, "%s:\n", be.directive.Position().Start)
	p.PrintDirective(&b, be.directive)
	fmt.Fprintf(&b, "\n%s\n", be.msg)
	return b.String()
}
