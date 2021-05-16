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
	"strings"
	"time"
	"unicode"

	"github.com/sboehler/knut/lib/date"
	"github.com/sboehler/knut/lib/ledger"
	"github.com/sboehler/knut/lib/model"
	"github.com/sboehler/knut/lib/scanner"
)

// Parser parses a journal
type Parser struct {
	scanner  *scanner.Scanner
	startPos model.FilePosition
}

func (p *Parser) markStart() {
	p.startPos = p.scanner.Position()
}

func (p *Parser) getRange() model.Range {
	var pos = p.scanner.Position()
	return model.Range{
		Start: p.startPos,
		End:   pos,
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
	d, err := scanner.ParseDate(p.scanner)
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

func (p *Parser) parseTransaction(d time.Time) (*ledger.Transaction, error) {
	desc, err := scanner.ReadQuotedString(p.scanner)
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
	return &ledger.Transaction{
		Pos:         p.getRange(),
		Date:        d,
		Description: desc,
		Tags:        tags,
		Postings:    postings,
	}, nil

}

func (p *Parser) parseAccrual() (*ledger.Accrual, error) {
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
		return nil, fmt.Errorf("expected \"once\", \"daily\", \"weekly\", \"monthly\", \"quarterly\" or \"yearly\", got %q", periodStr)
	}
	if err := p.consumeWhitespace1(); err != nil {
		return nil, err
	}
	dateFrom, err := scanner.ParseDate(p.scanner)
	if err != nil {
		return nil, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return nil, err
	}
	dateTo, err := scanner.ParseDate(p.scanner)
	if err != nil {
		return nil, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return nil, err
	}
	account, err := scanner.ParseAccount(p.scanner)
	if err != nil {
		return nil, err
	}
	if err := p.consumeRestOfWhitespaceLine(); err != nil {
		return nil, err
	}
	d, err := scanner.ParseDate(p.scanner)
	if err != nil {
		return nil, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return nil, err
	}
	t, err := p.parseTransaction(d)
	if err != nil {
		return nil, err
	}
	return &ledger.Accrual{
		Pos:         p.getRange(),
		T0:          dateFrom,
		T1:          dateTo,
		Period:      period,
		Account:     account,
		Transaction: t,
	}, nil
}

func (p *Parser) parsePostings() ([]*ledger.Posting, error) {
	var (
		postings []*ledger.Posting
		err      error
	)
	for !unicode.IsSpace(p.current()) && p.current() != scanner.EOF {
		var posting ledger.Posting
		if posting.Credit, err = scanner.ParseAccount(p.scanner); err != nil {
			return nil, err
		}
		if err = p.consumeWhitespace1(); err != nil {
			return nil, err
		}
		if posting.Debit, err = scanner.ParseAccount(p.scanner); err != nil {
			return nil, err
		}
		if err = p.consumeWhitespace1(); err != nil {
			return nil, err
		}
		if posting.Amount, err = scanner.ParseDecimal(p.scanner); err != nil {
			return nil, err
		}
		if err = p.consumeWhitespace1(); err != nil {
			return nil, err
		}
		if posting.Commodity, err = scanner.ParseCommodity(p.scanner); err != nil {
			return nil, err
		}
		if err = p.consumeWhitespace1(); err != nil {
			return nil, err
		}
		if unicode.IsLetter(p.current()) || unicode.IsDigit(p.current()) {
			if posting.Target, err = scanner.ParseCommodity(p.scanner); err != nil {
				return nil, err
			}
			if err = p.consumeWhitespace1(); err != nil {
				return nil, err
			}
		} else {
			posting.Target = posting.Commodity
		}
		if p.current() == '{' {
			if posting.Lot, err = p.parseLot(); err != nil {
				return nil, err
			}
			if err = p.consumeWhitespace1(); err != nil {
				return nil, err
			}
		}
		postings = append(postings, &posting)
		if err = p.consumeRestOfWhitespaceLine(); err != nil {
			return nil, err
		}
	}
	return postings, nil
}

func (p *Parser) parseOpen(d time.Time) (*ledger.Open, error) {
	if err := p.scanner.ParseString("open"); err != nil {
		return nil, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return nil, err
	}
	account, err := scanner.ParseAccount(p.scanner)
	if err != nil {
		return nil, err
	}
	return &ledger.Open{
		Pos:     p.getRange(),
		Date:    d,
		Account: account,
	}, nil
}

func (p *Parser) parseClose(d time.Time) (*ledger.Close, error) {
	if err := p.scanner.ParseString("close"); err != nil {
		return nil, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return nil, err
	}
	account, err := scanner.ParseAccount(p.scanner)
	if err != nil {
		return nil, err
	}
	return &ledger.Close{
		Pos:     p.getRange(),
		Date:    d,
		Account: account,
	}, nil
}

func (p *Parser) parsePrice(d time.Time) (*ledger.Price, error) {
	if err := p.scanner.ParseString("price"); err != nil {
		return nil, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return nil, err
	}
	commodity, err := scanner.ParseCommodity(p.scanner)
	if err != nil {
		return nil, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return nil, err
	}

	price, err := scanner.ParseDecimal(p.scanner)
	if err != nil {
		return nil, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return nil, err
	}
	target, err := scanner.ParseCommodity(p.scanner)
	if err != nil {
		return nil, err
	}
	return &ledger.Price{
		Pos:       p.getRange(),
		Date:      d,
		Commodity: commodity,
		Price:     price,
		Target:    target,
	}, nil
}

func (p *Parser) parseBalanceAssertion(d time.Time) (*ledger.Assertion, error) {
	if err := p.scanner.ParseString("balance"); err != nil {
		return nil, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return nil, err
	}
	account, err := scanner.ParseAccount(p.scanner)
	if err != nil {
		return nil, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return nil, err
	}
	amount, err := scanner.ParseDecimal(p.scanner)
	if err != nil {
		return nil, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return nil, err
	}
	commodity, err := scanner.ParseCommodity(p.scanner)
	if err != nil {
		return nil, err
	}
	return &ledger.Assertion{
		Pos:       p.getRange(),
		Date:      d,
		Account:   account,
		Amount:    amount,
		Commodity: commodity,
	}, nil
}

func (p *Parser) parseValue(d time.Time) (*ledger.Value, error) {
	if err := p.scanner.ParseString("value"); err != nil {
		return nil, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return nil, err
	}
	account, err := scanner.ParseAccount(p.scanner)
	if err != nil {
		return nil, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return nil, err
	}
	amount, err := scanner.ParseDecimal(p.scanner)
	if err != nil {
		return nil, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return nil, err
	}
	commodity, err := scanner.ParseCommodity(p.scanner)
	if err != nil {
		return nil, err
	}
	return &ledger.Value{
		Pos:       p.getRange(),
		Date:      d,
		Account:   account,
		Amount:    amount,
		Commodity: commodity,
	}, nil
}

func (p *Parser) parseInclude() (*ledger.Include, error) {
	p.markStart()
	if err := p.scanner.ParseString("include"); err != nil {
		return nil, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return nil, err
	}
	i, err := scanner.ReadQuotedString(p.scanner)
	if err != nil {
		return nil, err
	}
	result := &ledger.Include{
		Pos:  p.getRange(),
		Path: i,
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
	price, err := scanner.ParseFloat(p.scanner)
	if err != nil {
		return nil, err
	}
	if err := p.scanner.ConsumeWhile(isWhitespace); err != nil {
		return nil, err
	}
	commodity, err := scanner.ParseCommodity(p.scanner)
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
			if label, err = scanner.ReadQuotedString(p.scanner); err != nil {
				return nil, err
			}
			if err := p.scanner.ConsumeWhile(isWhitespace); err != nil {
				return nil, err
			}
		case unicode.IsDigit(p.current()):
			if d, err = scanner.ParseDate(p.scanner); err != nil {
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
	i, err := scanner.ParseIdentifier(p.scanner)
	if err != nil {
		return "", err
	}
	b.WriteString(i)
	return ledger.Tag(b.String()), nil
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
