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

	"github.com/sboehler/knut/lib/amount"
	"github.com/sboehler/knut/lib/ledger"
	"github.com/sboehler/knut/lib/model"
	"github.com/sboehler/knut/lib/scanner"
)

// parser parses a journal
type parser struct {
	scanner          *scanner.Scanner
	startPos, endPos model.FilePosition
}

func (p *parser) markStart() {
	p.startPos = p.scanner.Position()
}

func (p *parser) getRange() model.Range {
	pos := p.scanner.Position()
	return model.Range{
		Start: p.startPos,
		End:   pos,
	}
}

// new creates a new parser
func new(path string, r io.RuneReader) (*parser, error) {
	s, err := scanner.New(r, path)
	if err != nil {
		return nil, err
	}
	return &parser{scanner: s}, nil
}

// open creates a new parser for the given file.
func open(path string) (*parser, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return new(path, bufio.NewReader(f))
}

// current returns the current rune.
func (p *parser) current() rune {
	return p.scanner.Current()
}

// next returns the next directive
func (p *parser) next() (interface{}, error) {
	for p.current() != scanner.EOF {
		if err := p.scanner.ConsumeWhile(isWhitespaceOrNewline); err != nil {
			return nil, p.scanner.ParseError(err)
		}
		switch {
		case p.current() == '*' || p.current() == '#':
			if err := p.consumeComment(); err != nil {
				return nil, p.scanner.ParseError(err)
			}
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

func (p *parser) consumeComment() error {
	if err := p.scanner.ConsumeUntil(isNewline); err != nil {
		return err
	}
	if err := p.consumeNewline(); err != nil {
		return err
	}
	return nil
}

func (p *parser) parseDirective() (interface{}, error) {
	p.markStart()
	d, err := scanner.ParseDate(p.scanner)
	if err != nil {
		return nil, err
	}
	if err := p.consumeWhitespace1(); err != nil {
		return nil, err
	}
	var result interface{}
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

func (p *parser) parseTransaction(d time.Time) (*ledger.Transaction, error) {
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

func (p *parser) parsePostings() ([]*ledger.Posting, error) {
	var postings []*ledger.Posting
	for !unicode.IsSpace(p.current()) && p.current() != scanner.EOF {
		crAccount, err := scanner.ParseAccount(p.scanner)
		if err != nil {
			return nil, err
		}
		if err = p.consumeWhitespace1(); err != nil {
			return nil, err
		}
		drAccount, err := scanner.ParseAccount(p.scanner)
		if err != nil {
			return nil, err
		}
		if err = p.consumeWhitespace1(); err != nil {
			return nil, err
		}
		amt, err := scanner.ParseDecimal(p.scanner)
		if err != nil {
			return nil, err
		}
		if err = p.consumeWhitespace1(); err != nil {
			return nil, err
		}
		commodity, err := scanner.ParseCommodity(p.scanner)
		if err != nil {
			return nil, err
		}
		if err = p.consumeWhitespace1(); err != nil {
			return nil, err
		}
		var lot *ledger.Lot
		if p.current() == '{' {
			if lot, err = p.parseLot(); err != nil {
				return nil, err
			}
			if err = p.consumeWhitespace1(); err != nil {
				return nil, err
			}
		}
		var tag *ledger.Tag
		if p.current() == '#' {
			if tag, err = p.parseTag(); err != nil {
				return nil, err
			}
			if err = p.consumeWhitespace1(); err != nil {
				return nil, err
			}
		}
		postings = append(postings,
			&ledger.Posting{
				Amount:    amount.New(amt, nil),
				Credit:    crAccount,
				Debit:     drAccount,
				Commodity: commodity,
				Lot:       lot,
				Tag:       tag,
			},
		)
		if err = p.consumeRestOfWhitespaceLine(); err != nil {
			return nil, err
		}
	}
	return postings, nil
}

func (p *parser) parseOpen(d time.Time) (*ledger.Open, error) {
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

func (p *parser) parseClose(d time.Time) (*ledger.Close, error) {
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

func (p *parser) parsePrice(d time.Time) (*ledger.Price, error) {
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

	price, err := scanner.ParseFloat(p.scanner)
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
		Date:      d,
		Commodity: commodity,
		Price:     price,
		Target:    target,
	}, nil
}

func (p *parser) parseBalanceAssertion(d time.Time) (*ledger.Assertion, error) {
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

func (p *parser) parseValue(d time.Time) (*ledger.Value, error) {
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

func (p *parser) parseInclude() (*ledger.Include, error) {
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

func (p *parser) consumeNewline() error {
	if p.current() != scanner.EOF {
		return p.scanner.ConsumeRune('\n')
	}
	return nil
}

func (p *parser) consumeWhitespace1() error {
	if !isWhitespaceOrNewline(p.current()) && p.current() != scanner.EOF {
		return fmt.Errorf("expected whitespace, got %q", p.current())
	}
	return p.scanner.ConsumeWhile(isWhitespace)
}

func (p *parser) consumeRestOfWhitespaceLine() error {
	if err := p.consumeWhitespace1(); err != nil {
		return err
	}
	return p.consumeNewline()
}

func (p *parser) parseLot() (*ledger.Lot, error) {
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

func (p *parser) parseTags() ([]ledger.Tag, error) {
	var tags []ledger.Tag
	for p.current() == '#' {
		tag, err := p.parseTag()
		if err != nil {
			return nil, err
		}
		tags = append(tags, *tag)
		if err := p.consumeWhitespace1(); err != nil {
			return nil, err
		}
	}
	return tags, nil
}

func (p *parser) parseTag() (*ledger.Tag, error) {
	if p.current() != '#' {
		return nil, fmt.Errorf("Expected tag, got %c", p.current())
	}
	if err := p.scanner.ConsumeRune('#'); err != nil {
		return nil, err
	}
	b := strings.Builder{}
	b.WriteRune('#')
	i, err := scanner.ParseIdentifier(p.scanner)
	if err != nil {
		return nil, err
	}
	b.WriteString(i)
	tag := ledger.Tag(b.String())
	return &tag, nil
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
