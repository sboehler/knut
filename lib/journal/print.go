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

package journal

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
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
func (p Printer) PrintDirective(w io.Writer, directive Directive) (n int, err error) {
	switch d := directive.(type) {
	case *Transaction:
		return p.printTransaction(w, d)
	case *Open:
		return p.printOpen(w, d)
	case *Close:
		return p.printClose(w, d)
	case *Assertion:
		return p.printAssertion(w, d)
	case *Include:
		return p.printInclude(w, d)
	case *Price:
		return p.printPrice(w, d)
	case *Value:
		return p.printValue(w, d)
	}
	return 0, fmt.Errorf("unknown directive: %v", directive)
}

func (p Printer) printTransaction(w io.Writer, t *Transaction) (n int, err error) {
	if t.Accrual != nil {
		c, err := p.printAccrual(w, t.Accrual)
		n += c
		if err != nil {
			return n, err
		}
	}
	c, err := fmt.Fprintf(w, "%s \"%s\"", t.Date.Format("2006-01-02"), t.Description)
	n += c
	if err != nil {
		return n, err
	}
	for _, tag := range t.Tags {
		c, err := fmt.Fprintf(w, " %s", tag)
		n += c
		if err != nil {
			return n, err
		}
	}
	err = p.newline(w, &n)
	if err != nil {
		return n, err
	}
	for _, po := range t.Postings {
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

func (p Printer) printAccrual(w io.Writer, a *Accrual) (n int, err error) {
	return fmt.Fprintf(w, "@accrue %s %s %s %s\n", a.Interval, a.T0.Format("2006-01-02"), a.T1.Format("2006-01-02"), a.Account)
}

func (p Printer) printPosting(w io.Writer, t *Posting) (int, error) {
	var n int
	c, err := fmt.Fprintf(w, "%s %s %s %s", p.rightPad(t.Credit), p.rightPad(t.Debit), leftPad(10, t.Amount.String()), t.Commodity.Name())
	n += c
	if err != nil {
		return n, err
	}
	if t.Targets != nil {
		var s []string
		for _, t := range t.Targets {
			s = append(s, t.Name())
		}
		c, err = fmt.Fprintf(w, " (%s)", strings.Join(s, ","))
		n += c
		if err != nil {
			return n, err
		}
	}
	if t.Lot != nil {
		c, err = io.WriteString(w, " ")
		n += c
		if err != nil {
			return n, err
		}
		d, err := p.printLot(w, t.Lot)
		n += d
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

func (p Printer) printLot(w io.Writer, l *Lot) (int, error) {
	var n int
	c, err := fmt.Fprintf(w, "{ %g %s, %s ", l.Price, l.Commodity.Name(), l.Date.Format("2006-01-02"))
	n += c
	if err != nil {
		return n, err
	}
	if len(l.Label) > 0 {
		c, err = fmt.Fprintf(w, "%s ", l.Label)
		n += c
		if err != nil {
			return n, err
		}
	}
	c, err = io.WriteString(w, "}")
	n += c
	if err != nil {
		return n, err
	}
	return n, nil
}

func (p Printer) printOpen(w io.Writer, o *Open) (int, error) {
	return fmt.Fprintf(w, "%s open %s", o.Date.Format("2006-01-02"), o.Account)
}

func (p Printer) printClose(w io.Writer, c *Close) (int, error) {
	return fmt.Fprintf(w, "%s close %s", c.Date.Format("2006-01-02"), c.Account)
}

func (p Printer) printPrice(w io.Writer, pr *Price) (int, error) {
	return fmt.Fprintf(w, "%s price %s %s %s", pr.Date.Format("2006-01-02"), pr.Commodity.Name(), pr.Price, pr.Target.Name())
}

func (p Printer) printInclude(w io.Writer, i *Include) (int, error) {
	return fmt.Fprintf(w, "include \"%s\"", i.Path)
}

func (p Printer) printAssertion(w io.Writer, a *Assertion) (int, error) {
	return fmt.Fprintf(w, "%s balance %s %s %s", a.Date.Format("2006-01-02"), a.Account, a.Amount, a.Commodity.Name())
}

func (p Printer) printValue(w io.Writer, v *Value) (int, error) {
	return fmt.Fprintf(w, "%s value %s %s %s", v.Date.Format("2006-01-02"), v.Account, v.Amount, v.Commodity.Name())
}

// PrintLedger prints a Ledger.
func (p *Printer) PrintLedger(w io.Writer, l []*Day) (int, error) {
	for _, day := range l {
		for _, t := range day.Transactions {
			p.updatePadding(t)
		}
	}
	var n int
	for _, day := range l {
		for _, pr := range day.Prices {
			if err := p.writeLn(w, pr, &n); err != nil {
				return n, err
			}
		}
		if len(day.Prices) > 0 {
			if err := p.newline(w, &n); err != nil {
				return n, err
			}
		}
		for _, o := range day.Openings {
			if err := p.writeLn(w, o, &n); err != nil {
				return n, err
			}
		}
		if len(day.Openings) > 0 {
			if err := p.newline(w, &n); err != nil {
				return n, err
			}
		}
		for _, t := range day.Transactions {
			if err := p.writeLn(w, t, &n); err != nil {
				return n, err
			}
		}
		for _, v := range day.Values {
			if err := p.writeLn(w, v, &n); err != nil {
				return n, err
			}
		}
		if len(day.Values) > 0 {
			if err := p.newline(w, &n); err != nil {
				return n, err
			}
		}
		for _, a := range day.Assertions {
			if err := p.writeLn(w, a, &n); err != nil {
				return n, err
			}
		}
		if len(day.Assertions) > 0 {
			if err := p.newline(w, &n); err != nil {
				return n, err
			}
		}
		for _, c := range day.Closings {
			if err := p.writeLn(w, c, &n); err != nil {
				return n, err
			}
		}
		if len(day.Closings) > 0 {
			if err := p.newline(w, &n); err != nil {
				return n, err
			}
		}
	}
	return n, nil
}

// Initialize initializes the padding of this printer.
func (p *Printer) Initialize(directive []Directive) {
	for _, d := range directive {
		switch t := d.(type) {
		case *Transaction:
			p.updatePadding(t)
		}
	}
}

func (p *Printer) updatePadding(t *Transaction) {
	for _, pt := range t.Postings {
		var cr, dr = utf8.RuneCountInString(pt.Credit.String()), utf8.RuneCountInString(pt.Debit.String())
		if p.Padding < cr {
			p.Padding = cr
		}
		if p.Padding < dr {
			p.Padding = dr
		}
	}
}

func (p Printer) writeLn(w io.Writer, d Directive, count *int) error {
	c, err := p.PrintDirective(w, d)
	*count += c
	if err != nil {
		return err
	}
	return p.newline(w, count)
}

func (p Printer) newline(w io.Writer, count *int) error {
	c, err := io.WriteString(w, "\n")
	*count += c
	return err
}

func (p Printer) rightPad(a *Account) string {
	var b strings.Builder
	b.WriteString(a.String())
	for i := utf8.RuneCountInString(a.String()); i < p.Padding; i++ {
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
