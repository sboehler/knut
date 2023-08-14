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

	"github.com/sboehler/knut/lib/model"
)

// Printer prints directives.
type Printer struct {
	writer  io.Writer
	padding int
	count   int
}

// New creates a new Printer.
func New(w io.Writer) *Printer {
	return &Printer{writer: w}
}

func (p *Printer) Write(bs []byte) (int, error) {
	n, err := p.writer.Write(bs)
	p.count += n
	return n, err
}

// PrintDirective prints a directive to the given Writer.
func (p *Printer) PrintDirective(directive model.Directive) (n int, err error) {
	switch d := directive.(type) {
	case *model.Transaction:
		return p.printTransaction(d)
	case *model.Open:
		return p.printOpen(d)
	case *model.Close:
		return p.printClose(d)
	case *model.Assertion:
		return p.printAssertion(d)
	case *model.Price:
		return p.printPrice(d)
	}
	return 0, fmt.Errorf("unknown directive: %v", directive)
}

// PrintDirectiveLn prints a directive to the given Writer, followed by a newline.
func (p *Printer) PrintDirectiveLn(d model.Directive) (n int, err error) {
	start := p.count
	if _, err := p.PrintDirective(d); err != nil {
		return p.count - start, err
	}
	_, err = io.WriteString(p, "\n")
	return p.count - start, err
}

func (p *Printer) printTransaction(t *model.Transaction) (n int, err error) {
	start := p.count
	if t.Targets != nil {
		var s []string
		for _, t := range t.Targets {
			s = append(s, t.Name())
		}
		if _, err := fmt.Fprintf(p, "@performance(%s)\n", strings.Join(s, ",")); err != nil {
			return p.count - start, err
		}
	}
	if _, err := fmt.Fprintf(p, "%s \"%s\"", t.Date.Format("2006-01-02"), t.Description); err != nil {
		return p.count - start, err
	}
	if _, err := io.WriteString(p, "\n"); err != nil {
		return p.count - start, err
	}
	for i, po := range t.Postings {
		if i%2 == 0 {
			continue
		}
		if _, err := p.printPosting(po); err != nil {
			return p.count - start, err
		}
		if _, err := io.WriteString(p, "\n"); err != nil {
			return p.count - start, err
		}
	}
	return p.count - start, nil
}

func (p *Printer) printPosting(t *model.Posting) (int, error) {
	return fmt.Fprintf(p, "%-*s %-*s %10s %s", p.padding, t.Other.String(), p.padding, t.Account.String(), t.Quantity.String(), t.Commodity.Name())
}

func (p *Printer) printOpen(o *model.Open) (int, error) {
	return fmt.Fprintf(p, "%s open %s", o.Date.Format("2006-01-02"), o.Account)
}

func (p *Printer) printClose(c *model.Close) (int, error) {
	return fmt.Fprintf(p, "%s close %s", c.Date.Format("2006-01-02"), c.Account)
}

func (p *Printer) printPrice(pr *model.Price) (int, error) {
	return fmt.Fprintf(p, "%s price %s %s %s", pr.Date.Format("2006-01-02"), pr.Commodity.Name(), pr.Price, pr.Target.Name())
}

func (p *Printer) printAssertion(a *model.Assertion) (int, error) {
	return fmt.Fprintf(p, "%s balance %s %s %s", a.Date.Format("2006-01-02"), a.Account, a.Quantity, a.Commodity.Name())
}

// Initialize initializes the padding of this printer.
func (p *Printer) Initialize(directive []model.Directive) {
	for _, d := range directive {
		switch t := d.(type) {
		case *model.Transaction:
			p.UpdatePadding(t)
		}
	}
}

func (p *Printer) UpdatePadding(t *model.Transaction) {
	for _, pt := range t.Postings {
		cr, dr := utf8.RuneCountInString(pt.Account.String()), utf8.RuneCountInString(pt.Other.String())
		if p.padding < cr {
			p.padding = cr
		}
		if p.padding < dr {
			p.padding = dr
		}
	}
}
