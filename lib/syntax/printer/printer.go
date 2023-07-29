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

package printer

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/sboehler/knut/lib/syntax"
)

// Printer prints directives.
type Printer struct {
	Padding int
}

// New creates a new Printer.
func NewPrinter() *Printer {
	return new(Printer)
}

// PrintDirective prints a directive to the given Writer.
func (p Printer) PrintDirective(w io.Writer, directive syntax.Directive) (n int, err error) {
	switch d := directive.Directive.(type) {
	case syntax.Transaction:
		return p.printTransaction(w, d)
	case syntax.Open:
		return p.printOpen(w, d)
	case syntax.Close:
		return p.printClose(w, d)
	case syntax.Assertion:
		return p.printAssertion(w, d)
	case syntax.Include:
		return p.printInclude(w, d)
	case syntax.Price:
		return p.printPrice(w, d)
	}
	return 0, fmt.Errorf("unknown directive: %v", directive)
}

func (p Printer) printTransaction(w io.Writer, t syntax.Transaction) (n int, err error) {
	if !t.Addons.Accrual.Empty() {
		c, err := p.printAccrual(w, t.Addons.Accrual)
		n += c
		if err != nil {
			return n, err
		}
	}
	if !t.Addons.Performance.Empty() {
		var s []string
		for _, t := range t.Addons.Performance.Targets {
			s = append(s, t.Extract())
		}
		c, err := fmt.Fprintf(w, "@performance(%s)\n", strings.Join(s, ","))
		n += c
		if err != nil {
			return n, err
		}
	}
	c, err := fmt.Fprintf(w, `%s "%s"`, t.Date.Extract(), t.Description.Content.Extract())
	n += c
	if err != nil {
		return n, err
	}
	err = p.newline(w, &n)
	if err != nil {
		return n, err
	}
	for _, po := range t.Bookings {
		d, err := p.printPosting(w, po)
		n += d
		if err != nil {
			return n, err
		}
		err = p.newline(w, &n)
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

func (p Printer) printAccrual(w io.Writer, a syntax.Accrual) (n int, err error) {
	return fmt.Fprintf(w, "@accrue %s %s %s %s\n", a.Interval.Extract(), a.Start.Extract(), a.End.Extract(), a.Account.Extract())
}

func (p Printer) printPosting(w io.Writer, t syntax.Booking) (int, error) {
	var n int
	c, err := fmt.Fprintf(w, "%s %s %s %s", p.rightPad(t.Credit.Extract()), p.rightPad(t.Debit.Extract()), leftPad(10, t.Amount.Extract()), t.Commodity.Extract())
	n += c
	if err != nil {
		return n, err
	}
	return n, nil
}

func (p Printer) printOpen(w io.Writer, o syntax.Open) (int, error) {
	return fmt.Fprintf(w, "%s open %s", o.Date.Extract(), o.Account.Extract())
}

func (p Printer) printClose(w io.Writer, c syntax.Close) (int, error) {
	return fmt.Fprintf(w, "%s close %s", c.Date.Extract(), c.Account.Extract())
}

func (p Printer) printPrice(w io.Writer, pr syntax.Price) (int, error) {
	return fmt.Fprintf(w, "%s price %s %s %s", pr.Date.Extract(), pr.Commodity.Extract(), pr.Price.Extract(), pr.Target.Extract())
}

func (p Printer) printInclude(w io.Writer, i syntax.Include) (int, error) {
	return fmt.Fprintf(w, "include \"%s\"", i.Path.Extract())
}

func (p Printer) printAssertion(w io.Writer, a syntax.Assertion) (int, error) {
	return fmt.Fprintf(w, "%s balance %s %s %s", a.Date.Extract(), a.Account.Extract(), a.Amount.Extract(), a.Commodity.Extract())
}

// PrintLedger prints a Ledger.
func (p *Printer) PrintFile(w io.Writer, f syntax.File) (int, error) {
	var n int
	for _, d := range f.Directives {
		c, err := p.PrintDirective(w, d)
		n += c
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

// Initialize initializes the padding of this printer.
func (p *Printer) Initialize(directive []syntax.Directive) {
	for _, d := range directive {
		if t, ok := d.Directive.(syntax.Transaction); ok {
			for _, b := range t.Bookings {
				if p.Padding < b.Credit.Length() {
					p.Padding = b.Credit.Length()
				}
				if p.Padding < b.Debit.Length() {
					p.Padding = b.Debit.Length()
				}
			}
		}
	}
}

func (p Printer) newline(w io.Writer, count *int) error {
	c, err := io.WriteString(w, "\n")
	*count += c
	return err
}

func (p Printer) rightPad(s string) string {
	var b strings.Builder
	b.WriteString(s)
	for i := utf8.RuneCountInString(s); i < p.Padding; i++ {
		b.WriteRune(' ')
	}
	return b.String()
}

func leftPad(n int, s string) string {
	if len(s) > n {
		return s
	}
	var b strings.Builder
	for i := 0; i < n-len(s); i++ {
		b.WriteRune(' ')
	}
	b.WriteString(s)
	return b.String()
}
