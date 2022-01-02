package process

import (
	"fmt"
	"strings"

	"github.com/sboehler/knut/lib/journal/ast"
	"github.com/sboehler/knut/lib/journal/ast/printer"
)

// Error is an error.
type Error struct {
	directive ast.Directive
	msg       string
}

func (be Error) Error() string {
	var (
		p printer.Printer
		b strings.Builder
	)
	fmt.Fprintf(&b, "%s:\n", be.directive.Position().Start)
	p.PrintDirective(&b, be.directive)
	fmt.Fprintf(&b, "\n%s\n", be.msg)
	return b.String()
}
