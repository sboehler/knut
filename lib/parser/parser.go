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
	"github.com/sboehler/knut/lib/model"
	"github.com/sboehler/knut/lib/scanner"
)

// Parser parses a journal
type Parser struct {
	scanner          *scanner.Scanner
	startPos, endPos int
}

func (p *Parser) markStart() {
	p.startPos = p.scanner.Position
}

func (p *Parser) getRange() model.Range {
	return model.Range{Start: p.startPos, End: p.scanner.Position}
}

// New creates a new parser
func New(r io.RuneReader) (*Parser, error) {
	s, err := scanner.New(r)
	if err != nil {
		return nil, err
	}
	return &Parser{scanner: s}, nil
}

// Open creates a new parser for the given file.
func Open(path string) (*Parser, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return New(bufio.NewReader(f))
}

// current returns the current rune.
func (p *Parser) current() rune {
	return p.scanner.Current()
}

// next returns the next directive
func (p *Parser) next() (interface{}, error) {
	for p.current() != scanner.EOF {
		if err := p.scanner.ConsumeWhile(isWhitespaceOrNewline); err != nil {
			return nil, err
		}
		switch {
		case p.current() == '*' || p.current() == '#':
			if err := p.consumeComment(); err != nil {
				return nil, err
			}
		case p.current() == 'i':
			return p.parseInclude()
		case unicode.IsDigit(p.current()):
			return p.parseDirective()
		case p.current() != scanner.EOF:
			return nil, fmt.Errorf("%v: unexpected character: %v", p.scanner.Position, p.current())
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

func (p *Parser) parseDirective() (interface{}, error) {
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
	default:
		return nil, fmt.Errorf("expected directive, got %c", p.current())
	}
	if err != nil {
		return nil, err
	}
	if err := p.consumeRestOfWhitespaceLine(); err != nil {
		return nil, err
	}
	return result, nil
}

func (p *Parser) parseTransaction(d time.Time) (*model.Transaction, error) {
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
	return &model.Transaction{
		Directive:   model.NewDirective(p.getRange(), d),
		Description: desc,
		Tags:        tags,
		Postings:    postings,
	}, nil

}

func (p *Parser) parsePostings() ([]*model.Posting, error) {
	var postings []*model.Posting
	for unicode.IsLetter(p.current()) {
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
		var lot *model.Lot
		if p.current() == '{' {
			if lot, err = p.parseLot(); err != nil {
				return nil, err
			}
			if err = p.consumeWhitespace1(); err != nil {
				return nil, err
			}
		}
		var tag *model.Tag
		if p.current() == '#' {
			if tag, err = p.parseTag(); err != nil {
				return nil, err
			}
			if err = p.consumeWhitespace1(); err != nil {
				return nil, err
			}
		}
		postings = append(postings,
			&model.Posting{
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

func (p *Parser) parseOpen(d time.Time) (*model.Open, error) {
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
	return &model.Open{
		Directive: model.NewDirective(p.getRange(), d),
		Account:   account,
	}, nil
}

func (p *Parser) parseClose(d time.Time) (*model.Close, error) {
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
	return &model.Close{
		Directive: model.NewDirective(p.getRange(), d),
		Account:   account,
	}, nil
}

func (p *Parser) parsePrice(d time.Time) (*model.Price, error) {
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
	return &model.Price{
		Directive: model.NewDirective(p.getRange(), d),
		Commodity: commodity,
		Price:     price,
		Target:    target,
	}, nil
}

func (p *Parser) parseBalanceAssertion(d time.Time) (*model.Assertion, error) {
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
	return &model.Assertion{
		Directive: model.NewDirective(p.getRange(), d),
		Account:   account,
		Amount:    amount,
		Commodity: commodity,
	}, nil
}

func (p *Parser) parseInclude() (*model.Include, error) {
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
	result := &model.Include{
		Directive: model.NewDirective(p.getRange(), time.Time{}),
		Path:      i,
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
		return fmt.Errorf("expected whitespace, got %c", p.current())
	}
	return p.scanner.ConsumeWhile(isWhitespace)
}

func (p *Parser) consumeRestOfWhitespaceLine() error {
	if err := p.consumeWhitespace1(); err != nil {
		return err
	}
	return p.consumeNewline()
}

func (p *Parser) parseLot() (*model.Lot, error) {
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
			return nil, fmt.Errorf("expected label or date, got %v", p.current())
		}
	}
	err = p.scanner.ConsumeRune('}')
	if err != nil {
		return nil, err
	}
	return &model.Lot{
		Date:      d,
		Label:     label,
		Price:     price,
		Commodity: commodity,
	}, nil
}

func (p *Parser) parseTags() ([]model.Tag, error) {
	var tags []model.Tag
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

func (p *Parser) parseTag() (*model.Tag, error) {
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
	tag := model.Tag(b.String())
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
