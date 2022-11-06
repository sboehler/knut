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

package parser

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/sboehler/knut/lib/common/date"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/scanner"
	"github.com/shopspring/decimal"
)

// Parser parses a journal
type Parser struct {
	context  journal.Context
	scanner  *scanner.Scanner
	startPos scanner.Location
}

func (p *Parser) markStart() {
	p.startPos = p.scanner.Location
}

func (p *Parser) getRange() journal.Range {
	return journal.Range{
		Start: p.startPos,
		End:   p.scanner.Location,
		Path:  p.scanner.Path,
	}
}

// New creates a new parser
func New(ctx journal.Context, path string, r io.RuneReader) (*Parser, error) {
	s, err := scanner.New(r, path)
	if err != nil {
		return nil, err
	}
	return &Parser{
		context: ctx,
		scanner: s,
	}, nil
}

// FromPath creates a new parser for the given file.
func FromPath(ctx journal.Context, path string) (*Parser, func() error, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	p, err := New(ctx, path, bufio.NewReader(f))
	if err != nil {
		return nil, nil, err
	}
	return p, f.Close, nil
}

// current returns the current rune.
func (p *Parser) current() rune {
	return p.scanner.Current()
}

// Next returns the Next directive
func (p *Parser) Next() (journal.Directive, error) {
	for p.current() != scanner.EOF {
		if err := p.scanner.ConsumeWhile(isWhitespaceOrNewline); err != nil {
			return nil, p.scanner.ParseError(err)
		}
		switch {
		case p.current() == '*' || p.current() == '#':
			if err := p.consumeComment(); err != nil {
				return nil, p.scanner.ParseError(err)
			}
		case p.current() == '@':
			a, err := p.parseAddOn()
			if err != nil {
				return nil, p.scanner.ParseError(err)
			}
			d, err := p.parseDirective(a)
			if err != nil {
				return nil, p.scanner.ParseError(err)
			}
			return d, nil
		case p.current() == 'i':
			i, err := p.parseInclude()
			if err != nil {
				return nil, p.scanner.ParseError(err)
			}
			return i, nil
		case p.current() == 'c':
			c, err := p.parseCurrency()
			if err != nil {
				return nil, p.scanner.ParseError(err)
			}
			return c, nil
		case unicode.IsDigit(p.current()):
			d, err := p.parseDirective(nil)
			if err != nil {
				return nil, p.scanner.ParseError(err)
			}
			return d, nil
		case p.current() != scanner.EOF:
			return nil, p.scanner.ParseError(fmt.Errorf("unexpected character: %q", p.current()))
		}
	}
	return nil, io.EOF
}

func (p *Parser) consumeComment() error {
	if err := p.scanner.ConsumeUntil(isNewline); err != nil {
		return err
	}
	if err := p.consumeNewline(); err != nil {
		return err
	}
	return nil
}

func (p *Parser) parseDirective(a *journal.Accrual) (journal.Directive, error) {
	p.markStart()
	d, err := p.parseDate()
	if err != nil {
		return nil, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return nil, err
	}
	var result journal.Directive
	switch p.current() {
	case '"':
		result, err = p.parseTransaction(d, a)
	case 'o':
		result, err = p.parseOpen(d)
	case 'c':
		result, err = p.parseClose(d)
	case 'p':
		result, err = p.parsePrice(d)
	case 'b':
		result, err = p.parseBalanceAssertion(d)
	case 'v':
		result, err = p.parseValue(d)
	default:
		return nil, fmt.Errorf("expected directive, got %q", p.current())
	}
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (p *Parser) parseTransaction(d time.Time, a *journal.Accrual) (*journal.Transaction, error) {
	desc, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}

	if err := p.consumeWhitespace1(); err != nil {
		return nil, err
	}

	tags, err := p.parseTags()
	if err != nil {
		return nil, err
	}
	if err := p.consumeRestOfWhitespaceLine(); err != nil {
		return nil, err
	}
	postings, err := p.parsePostings()
	if err != nil {
		return nil, err
	}
	r := p.getRange()
	if a != nil {
		r.Start = a.Range.Start
	}
	return journal.TransactionBuilder{
		Range:       r,
		Date:        d,
		Description: desc,
		Tags:        tags,
		Postings:    postings,
		Accrual:     a,
	}.Build(), nil

}

func (p *Parser) parseAddOn() (*journal.Accrual, error) {
	p.markStart()
	if err := p.scanner.ConsumeRune('@'); err != nil {
		return nil, err
	}
	if err := p.scanner.ParseString("accrue"); err != nil {
		return nil, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return nil, err
	}
	periodStr, err := p.scanner.ReadWhile(unicode.IsLetter)
	if err != nil {
		return nil, err
	}
	var interval date.Interval
	switch periodStr {
	case "once":
		interval = date.Once
	case "daily":
		interval = date.Daily
	case "weekly":
		interval = date.Weekly
	case "monthly":
		interval = date.Monthly
	case "quarterly":
		interval = date.Quarterly
	case "yearly":
		interval = date.Yearly
	default:
		return nil, fmt.Errorf("expected \"once\", \"daily\", \"weekly\", \"monthly\", \"quarterly\" or \"yearly\", got %q", periodStr)
	}
	if err := p.consumeWhitespace1(); err != nil {
		return nil, err
	}
	dateFrom, err := p.parseDate()
	if err != nil {
		return nil, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return nil, err
	}
	dateTo, err := p.parseDate()
	if err != nil {
		return nil, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return nil, err
	}
	account, err := p.parseAccount()
	if err != nil {
		return nil, err
	}
	if err := p.consumeRestOfWhitespaceLine(); err != nil {
		return nil, err
	}
	return &journal.Accrual{
		Range:    p.getRange(),
		T0:       dateFrom,
		T1:       dateTo,
		Interval: interval,
		Account:  account,
	}, nil
}

func (p *Parser) parsePostings() ([]journal.Posting, error) {
	var postings []journal.Posting
	for !unicode.IsSpace(p.current()) && p.current() != scanner.EOF {
		var (
			credit, debit *journal.Account
			amount        decimal.Decimal
			commodity     *journal.Commodity
			targets       []*journal.Commodity
			lot           *journal.Lot

			err error
		)
		if credit, err = p.parseAccount(); err != nil {
			return nil, err
		}
		if err = p.consumeWhitespace1(); err != nil {
			return nil, err
		}
		if debit, err = p.parseAccount(); err != nil {
			return nil, err
		}
		if err = p.consumeWhitespace1(); err != nil {
			return nil, err
		}
		if amount, err = p.parseDecimal(); err != nil {
			return nil, err
		}
		if err = p.consumeWhitespace1(); err != nil {
			return nil, err
		}
		if commodity, err = p.parseCommodity(); err != nil {
			return nil, err
		}
		if err = p.consumeWhitespace1(); err != nil {
			return nil, err
		}
		for p.current() == '{' || p.current() == '(' {
			switch p.current() {
			case '{':
				if lot != nil {
					return nil, fmt.Errorf("duplicate lot")
				}
				if lot, err = p.parseLot(); err != nil {
					return nil, err
				}
				if err = p.consumeWhitespace1(); err != nil {
					return nil, err
				}
			case '(':
				if targets != nil {
					return nil, fmt.Errorf("duplicate target commodity declarations")
				}
				if targets, err = p.parseTargetCommodities(); err != nil {
					return nil, err
				}
				if err = p.consumeWhitespace1(); err != nil {
					return nil, err
				}
			}
		}
		postings = append(postings, journal.Posting{
			Credit:    credit,
			Debit:     debit,
			Amount:    amount,
			Commodity: commodity,
			Targets:   targets,
			Lot:       lot,
		})
		if err = p.consumeRestOfWhitespaceLine(); err != nil {
			return nil, err
		}
	}
	return postings, nil
}

func (p *Parser) parseOpen(d time.Time) (*journal.Open, error) {
	if err := p.scanner.ParseString("open"); err != nil {
		return nil, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return nil, err
	}
	account, err := p.parseAccount()
	if err != nil {
		return nil, err
	}
	return &journal.Open{
		Range:   p.getRange(),
		Date:    d,
		Account: account,
	}, nil
}

func (p *Parser) parseClose(d time.Time) (*journal.Close, error) {
	if err := p.scanner.ParseString("close"); err != nil {
		return nil, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return nil, err
	}
	account, err := p.parseAccount()
	if err != nil {
		return nil, err
	}
	return &journal.Close{
		Range:   p.getRange(),
		Date:    d,
		Account: account,
	}, nil
}

func (p *Parser) parsePrice(d time.Time) (*journal.Price, error) {
	if err := p.scanner.ParseString("price"); err != nil {
		return nil, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return nil, err
	}
	commodity, err := p.parseCommodity()
	if err != nil {
		return nil, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return nil, err
	}

	price, err := p.parseDecimal()
	if err != nil {
		return nil, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return nil, err
	}
	target, err := p.parseCommodity()
	if err != nil {
		return nil, err
	}
	return &journal.Price{
		Range:     p.getRange(),
		Date:      d,
		Commodity: commodity,
		Price:     price,
		Target:    target,
	}, nil
}

func (p *Parser) parseBalanceAssertion(d time.Time) (*journal.Assertion, error) {
	if err := p.scanner.ParseString("balance"); err != nil {
		return nil, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return nil, err
	}
	account, err := p.parseAccount()
	if err != nil {
		return nil, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return nil, err
	}
	amount, err := p.parseDecimal()
	if err != nil {
		return nil, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return nil, err
	}
	commodity, err := p.parseCommodity()
	if err != nil {
		return nil, err
	}
	return &journal.Assertion{
		Range:     p.getRange(),
		Date:      d,
		Account:   account,
		Amount:    amount,
		Commodity: commodity,
	}, nil
}

func (p *Parser) parseValue(d time.Time) (*journal.Value, error) {
	if err := p.scanner.ParseString("value"); err != nil {
		return nil, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return nil, err
	}
	account, err := p.parseAccount()
	if err != nil {
		return nil, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return nil, err
	}
	amount, err := p.parseDecimal()
	if err != nil {
		return nil, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return nil, err
	}
	commodity, err := p.parseCommodity()
	if err != nil {
		return nil, err
	}
	return &journal.Value{
		Range:     p.getRange(),
		Date:      d,
		Account:   account,
		Amount:    amount,
		Commodity: commodity,
	}, nil
}

func (p *Parser) parseInclude() (*journal.Include, error) {
	p.markStart()
	if err := p.scanner.ParseString("include"); err != nil {
		return nil, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return nil, err
	}
	i, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}
	result := &journal.Include{
		Range: p.getRange(),
		Path:  i,
	}
	if err := p.consumeRestOfWhitespaceLine(); err != nil {
		return nil, err
	}
	return result, nil
}

func (p *Parser) parseCurrency() (*journal.Currency, error) {
	p.markStart()
	if err := p.scanner.ParseString("currency"); err != nil {
		return nil, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return nil, err
	}
	i, err := p.parseCommodity()
	if err != nil {
		return nil, err
	}
	result := &journal.Currency{
		Range:     p.getRange(),
		Commodity: i,
	}
	if err := p.consumeRestOfWhitespaceLine(); err != nil {
		return nil, err
	}
	return result, nil
}

func (p *Parser) consumeNewline() error {
	if p.current() != scanner.EOF {
		return p.scanner.ConsumeRune('\n')
	}
	return nil
}

func (p *Parser) parseAccount() (*journal.Account, error) {
	s, err := p.scanner.ReadWhile(func(r rune) bool {
		return r == ':' || unicode.IsLetter(r) || unicode.IsDigit(r)
	})
	if err != nil {
		return nil, err
	}
	return p.context.GetAccount(s)
}

func (p *Parser) consumeWhitespace1() error {
	if !isWhitespaceOrNewline(p.current()) && p.current() != scanner.EOF {
		return fmt.Errorf("expected whitespace, got %q", p.current())
	}
	return p.scanner.ConsumeWhile(isWhitespace)
}

func (p *Parser) consumeRestOfWhitespaceLine() error {
	if err := p.consumeWhitespace1(); err != nil {
		return err
	}
	return p.consumeNewline()
}

func (p *Parser) parseLot() (*journal.Lot, error) {
	err := p.scanner.ConsumeRune('{')
	if err != nil {
		return nil, err
	}
	if err := p.scanner.ConsumeWhile(isWhitespace); err != nil {
		return nil, err
	}
	price, err := p.parseFloat()
	if err != nil {
		return nil, err
	}
	if err := p.scanner.ConsumeWhile(isWhitespace); err != nil {
		return nil, err
	}
	commodity, err := p.parseCommodity()
	if err != nil {
		return nil, err
	}
	if err := p.scanner.ConsumeWhile(isWhitespace); err != nil {
		return nil, err
	}
	var (
		label string
		d     time.Time
	)
	for p.current() == ',' {
		if err := p.scanner.ConsumeRune(','); err != nil {
			return nil, err
		}
		if err := p.scanner.ConsumeWhile(isWhitespace); err != nil {
			return nil, err
		}
		switch {
		case p.current() == '"':
			if label, err = p.parseQuotedString(); err != nil {
				return nil, err
			}
			if err := p.scanner.ConsumeWhile(isWhitespace); err != nil {
				return nil, err
			}
		case unicode.IsDigit(p.current()):
			if d, err = p.parseDate(); err != nil {
				return nil, err
			}
			if err := p.scanner.ConsumeWhile(isWhitespace); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("expected label or date, got %q", p.current())
		}
	}
	err = p.scanner.ConsumeRune('}')
	if err != nil {
		return nil, err
	}
	return &journal.Lot{
		Date:      d,
		Label:     label,
		Price:     price,
		Commodity: commodity,
	}, nil
}

func (p *Parser) parseTargetCommodities() ([]*journal.Commodity, error) {
	// we use non-nil slices of size 0 to mark portfolio income / expenses
	res := make([]*journal.Commodity, 0)
	if err := p.scanner.ConsumeRune('('); err != nil {
		return nil, err
	}
	if err := p.scanner.ConsumeWhile(isWhitespace); err != nil {
		return nil, err
	}
	if p.current() != ')' {
		c, err := p.parseCommodity()
		if err != nil {
			return nil, err
		}
		res = append(res, c)
		if err := p.scanner.ConsumeWhile(isWhitespace); err != nil {
			return nil, err
		}
	}
	for p.current() == ',' {
		if err := p.scanner.ConsumeRune(','); err != nil {
			return nil, err
		}
		if err := p.scanner.ConsumeWhile(isWhitespace); err != nil {
			return nil, err
		}
		c, err := p.parseCommodity()
		if err != nil {
			return nil, err
		}
		res = append(res, c)
		if err := p.scanner.ConsumeWhile(isWhitespace); err != nil {
			return nil, err
		}
	}
	if err := p.scanner.ConsumeRune(')'); err != nil {
		return nil, err
	}
	return res, nil
}

func (p *Parser) parseTags() ([]journal.Tag, error) {
	var tags []journal.Tag
	for p.current() == '#' {
		tag, err := p.parseTag()
		if err != nil {
			return nil, err
		}
		tags = append(tags, tag)
		if err := p.consumeWhitespace1(); err != nil {
			return nil, err
		}
	}
	return tags, nil
}

func (p *Parser) parseTag() (journal.Tag, error) {
	if p.current() != '#' {
		return "", fmt.Errorf("expected tag, got %c", p.current())
	}
	if err := p.scanner.ConsumeRune('#'); err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteRune('#')
	i, err := p.parseIdentifier()
	if err != nil {
		return "", err
	}
	b.WriteString(i)
	return journal.Tag(b.String()), nil
}

// parseQuotedString parses a quoted string
func (p *Parser) parseQuotedString() (string, error) {
	if err := p.scanner.ConsumeRune('"'); err != nil {
		return "", err
	}
	s, err := p.scanner.ReadWhile(func(r rune) bool {
		return r != '"'
	})
	if err != nil {
		return s, err
	}
	if err := p.scanner.ConsumeRune('"'); err != nil {
		return s, err
	}
	return s, nil
}

// parseIdentifier parses an identifier
func (p *Parser) parseIdentifier() (string, error) {
	var s strings.Builder
	if !(unicode.IsLetter(p.scanner.Current()) || unicode.IsDigit(p.scanner.Current())) {
		return "", fmt.Errorf("expected identifier, got %q", p.scanner.Current())
	}
	for unicode.IsLetter(p.scanner.Current()) || unicode.IsDigit(p.scanner.Current()) {
		s.WriteRune(p.scanner.Current())
		if err := p.scanner.Advance(); err != nil {
			return s.String(), err
		}
	}
	return s.String(), nil
}

// parseDecimal parses a decimal number
func (p *Parser) parseDecimal() (decimal.Decimal, error) {
	var b strings.Builder
	for unicode.IsDigit(p.scanner.Current()) || p.scanner.Current() == '.' || p.scanner.Current() == '-' {
		b.WriteRune(p.scanner.Current())
		if err := p.scanner.Advance(); err != nil {
			return decimal.Zero, err
		}
	}
	return decimal.NewFromString(b.String())
}

// parseDate parses a date as YYYY-MM-DD
func (p *Parser) parseDate() (time.Time, error) {
	d, err := p.scanner.ReadN(10)
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse("2006-01-02", d)
}

// parseFloat parses a floating point number
func (p *Parser) parseFloat() (float64, error) {
	var b strings.Builder
	for unicode.IsDigit(p.scanner.Current()) || p.scanner.Current() == '.' || p.scanner.Current() == '-' {
		b.WriteRune(p.scanner.Current())
		if err := p.scanner.Advance(); err != nil {
			return 0, err
		}
	}
	return strconv.ParseFloat(b.String(), 64)
}

// parseCommodity parses a commodity
func (p *Parser) parseCommodity() (*journal.Commodity, error) {
	i, err := p.parseIdentifier()
	if err != nil {
		return nil, err
	}
	return p.context.GetCommodity(i)
}
func isWhitespace(ch rune) bool {
	return ch == ' ' || ch == '\t' || ch == '\r'
}

func isNewline(ch rune) bool {
	return ch == '\n'
}

func isWhitespaceOrNewline(ch rune) bool {
	return isNewline(ch) || isWhitespace(ch)
}
