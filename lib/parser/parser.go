// Copyright 2020 Silvio Böhler
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

	"github.com/sboehler/knut/lib/date"
	"github.com/sboehler/knut/lib/ledger"
	"github.com/sboehler/knut/lib/model/accounts"
	"github.com/sboehler/knut/lib/model/commodities"
	"github.com/sboehler/knut/lib/scanner"
	"github.com/shopspring/decimal"
)

// Parser parses a journal
type Parser struct {
	scanner  *scanner.Scanner
	startPos scanner.Location
}

func (p *Parser) markStart() {
	p.startPos = p.scanner.Location
}

func (p *Parser) getRange() ledger.Range {
	return ledger.Range{
		Start: p.startPos,
		End:   p.scanner.Location,
		Path:  p.scanner.Path,
	}
}

// New creates a new parser
func New(path string, r io.RuneReader) (*Parser, error) {
	s, err := scanner.New(r, path)
	if err != nil {
		return nil, err
	}
	return &Parser{scanner: s}, nil
}

// FromPath creates a new parser for the given file.
func FromPath(path string) (*Parser, func() error, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	p, err := New(path, bufio.NewReader(f))
	if err != nil {
		return nil, nil, err
	}
	return p, f.Close, nil
}

// current returns the current rune.
func (p *Parser) current() rune {
	return p.scanner.Current()
}

// Next returns the next directive
func (p *Parser) Next() (ledger.Directive, error) {
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
			a, err := p.parseAccrual()
			if err != nil {
				return nil, p.scanner.ParseError(err)
			}
			return a, nil
		case p.current() == 'i':
			i, err := p.parseInclude()
			if err != nil {
				return nil, p.scanner.ParseError(err)
			}
			return i, nil
		case unicode.IsDigit(p.current()):
			d, err := p.parseDirective()
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

// ParseAll parses the entire stream asynchronously.
func (p *Parser) ParseAll() <-chan interface{} {
	var ch = make(chan interface{}, 10)
	go func() {
		defer close(ch)
		for {
			d, err := p.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				ch <- err
				break
			}
			ch <- d
		}
	}()
	return ch
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

func (p *Parser) parseDirective() (ledger.Directive, error) {
	p.markStart()
	d, err := p.parseDate()
	if err != nil {
		return nil, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return nil, err
	}
	var result ledger.Directive
	switch p.current() {
	case '"':
		result, err = p.parseTransaction(d)
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

func (p *Parser) parseTransaction(d time.Time) (ledger.Transaction, error) {
	desc, err := p.parseQuotedString()
	if err != nil {
		return ledger.Transaction{}, err
	}

	if err := p.consumeWhitespace1(); err != nil {
		return ledger.Transaction{}, err
	}

	tags, err := p.parseTags()
	if err != nil {
		return ledger.Transaction{}, err
	}
	if err := p.consumeRestOfWhitespaceLine(); err != nil {
		return ledger.Transaction{}, err
	}
	postings, err := p.parsePostings()
	if err != nil {
		return ledger.Transaction{}, err
	}
	return ledger.Transaction{
		Range:       p.getRange(),
		Date:        d,
		Description: desc,
		Tags:        tags,
		Postings:    postings,
	}, nil

}

func (p *Parser) parseAccrual() (ledger.Accrual, error) {
	p.markStart()
	if err := p.scanner.ConsumeRune('@'); err != nil {
		return ledger.Accrual{}, err
	}
	if err := p.scanner.ParseString("accrue"); err != nil {
		return ledger.Accrual{}, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return ledger.Accrual{}, err
	}
	periodStr, err := p.scanner.ReadWhile(unicode.IsLetter)
	if err != nil {
		return ledger.Accrual{}, err
	}
	var period date.Period
	switch periodStr {
	case "once":
		period = date.Once
	case "daily":
		period = date.Daily
	case "weekly":
		period = date.Weekly
	case "monthly":
		period = date.Monthly
	case "quarterly":
		period = date.Quarterly
	case "yearly":
		period = date.Yearly
	default:
		return ledger.Accrual{}, fmt.Errorf("expected \"once\", \"daily\", \"weekly\", \"monthly\", \"quarterly\" or \"yearly\", got %q", periodStr)
	}
	if err := p.consumeWhitespace1(); err != nil {
		return ledger.Accrual{}, err
	}
	dateFrom, err := p.parseDate()
	if err != nil {
		return ledger.Accrual{}, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return ledger.Accrual{}, err
	}
	dateTo, err := p.parseDate()
	if err != nil {
		return ledger.Accrual{}, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return ledger.Accrual{}, err
	}
	account, err := p.parseAccount()
	if err != nil {
		return ledger.Accrual{}, err
	}
	if err := p.consumeRestOfWhitespaceLine(); err != nil {
		return ledger.Accrual{}, err
	}
	d, err := p.parseDate()
	if err != nil {
		return ledger.Accrual{}, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return ledger.Accrual{}, err
	}
	t, err := p.parseTransaction(d)
	if err != nil {
		return ledger.Accrual{}, err
	}
	if len(t.Postings) != 1 {
		return ledger.Accrual{}, fmt.Errorf("accrual transaction must have exactly one posting: %v", t)
	}
	return ledger.Accrual{
		Range:       p.getRange(),
		T0:          dateFrom,
		T1:          dateTo,
		Period:      period,
		Account:     account,
		Transaction: t,
	}, nil
}

func (p *Parser) parsePostings() ([]ledger.Posting, error) {
	var postings []ledger.Posting
	for !unicode.IsSpace(p.current()) && p.current() != scanner.EOF {
		var (
			credit, debit *accounts.Account
			amount        decimal.Decimal
			commodity     *commodities.Commodity
			lot           *ledger.Lot

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
		if p.current() == '{' {
			if lot, err = p.parseLot(); err != nil {
				return nil, err
			}
			if err = p.consumeWhitespace1(); err != nil {
				return nil, err
			}
		}
		postings = append(postings, ledger.Posting{
			Credit:    credit,
			Debit:     debit,
			Amount:    amount,
			Commodity: commodity,
			Lot:       lot,
		})
		if err = p.consumeRestOfWhitespaceLine(); err != nil {
			return nil, err
		}
	}
	return postings, nil
}

func (p *Parser) parseOpen(d time.Time) (ledger.Open, error) {
	if err := p.scanner.ParseString("open"); err != nil {
		return ledger.Open{}, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return ledger.Open{}, err
	}
	account, err := p.parseAccount()
	if err != nil {
		return ledger.Open{}, err
	}
	return ledger.Open{
		Range:   p.getRange(),
		Date:    d,
		Account: account,
	}, nil
}

func (p *Parser) parseClose(d time.Time) (ledger.Close, error) {
	if err := p.scanner.ParseString("close"); err != nil {
		return ledger.Close{}, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return ledger.Close{}, err
	}
	account, err := p.parseAccount()
	if err != nil {
		return ledger.Close{}, err
	}
	return ledger.Close{
		Range:   p.getRange(),
		Date:    d,
		Account: account,
	}, nil
}

func (p *Parser) parsePrice(d time.Time) (ledger.Price, error) {
	if err := p.scanner.ParseString("price"); err != nil {
		return ledger.Price{}, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return ledger.Price{}, err
	}
	commodity, err := p.parseCommodity()
	if err != nil {
		return ledger.Price{}, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return ledger.Price{}, err
	}

	price, err := p.parseDecimal()
	if err != nil {
		return ledger.Price{}, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return ledger.Price{}, err
	}
	target, err := p.parseCommodity()
	if err != nil {
		return ledger.Price{}, err
	}
	return ledger.Price{
		Range:     p.getRange(),
		Date:      d,
		Commodity: commodity,
		Price:     price,
		Target:    target,
	}, nil
}

func (p *Parser) parseBalanceAssertion(d time.Time) (ledger.Assertion, error) {
	if err := p.scanner.ParseString("balance"); err != nil {
		return ledger.Assertion{}, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return ledger.Assertion{}, err
	}
	account, err := p.parseAccount()
	if err != nil {
		return ledger.Assertion{}, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return ledger.Assertion{}, err
	}
	amount, err := p.parseDecimal()
	if err != nil {
		return ledger.Assertion{}, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return ledger.Assertion{}, err
	}
	commodity, err := p.parseCommodity()
	if err != nil {
		return ledger.Assertion{}, err
	}
	return ledger.Assertion{
		Range:     p.getRange(),
		Date:      d,
		Account:   account,
		Amount:    amount,
		Commodity: commodity,
	}, nil
}

func (p *Parser) parseValue(d time.Time) (ledger.Value, error) {
	if err := p.scanner.ParseString("value"); err != nil {
		return ledger.Value{}, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return ledger.Value{}, err
	}
	account, err := p.parseAccount()
	if err != nil {
		return ledger.Value{}, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return ledger.Value{}, err
	}
	amount, err := p.parseDecimal()
	if err != nil {
		return ledger.Value{}, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return ledger.Value{}, err
	}
	commodity, err := p.parseCommodity()
	if err != nil {
		return ledger.Value{}, err
	}
	return ledger.Value{
		Range:     p.getRange(),
		Date:      d,
		Account:   account,
		Amount:    amount,
		Commodity: commodity,
	}, nil
}

func (p *Parser) parseInclude() (ledger.Include, error) {
	p.markStart()
	if err := p.scanner.ParseString("include"); err != nil {
		return ledger.Include{}, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return ledger.Include{}, err
	}
	i, err := p.parseQuotedString()
	if err != nil {
		return ledger.Include{}, err
	}
	result := ledger.Include{
		Range: p.getRange(),
		Path:  i,
	}
	if err := p.consumeRestOfWhitespaceLine(); err != nil {
		return ledger.Include{}, err
	}
	return result, nil
}

func (p *Parser) consumeNewline() error {
	if p.current() != scanner.EOF {
		return p.scanner.ConsumeRune('\n')
	}
	return nil
}

func (p *Parser) parseAccount() (*accounts.Account, error) {
	s, err := p.scanner.ReadWhile(func(r rune) bool {
		return r == ':' || unicode.IsLetter(r) || unicode.IsDigit(r)
	})
	if err != nil {
		return nil, err
	}
	return accounts.Get(s)
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

func (p *Parser) parseLot() (*ledger.Lot, error) {
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
	return &ledger.Lot{
		Date:      d,
		Label:     label,
		Price:     price,
		Commodity: commodity,
	}, nil
}

func (p *Parser) parseTags() ([]ledger.Tag, error) {
	var tags []ledger.Tag
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

func (p *Parser) parseTag() (ledger.Tag, error) {
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
	return ledger.Tag(b.String()), nil
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
func (p *Parser) parseCommodity() (*commodities.Commodity, error) {
	i, err := p.parseIdentifier()
	if err != nil {
		return nil, err
	}
	return commodities.Get(i)
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
