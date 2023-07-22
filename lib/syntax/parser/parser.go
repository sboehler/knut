package parser

import (
	"unicode"

	"github.com/sboehler/knut/lib/syntax"
	"github.com/sboehler/knut/lib/syntax/scanner"
)

// Parser parses a journal.
type Parser struct {
	scanner.Scanner
}

// New creates a new parser.
func New(text, path string) *Parser {
	return &Parser{
		Scanner: *scanner.New(string(text), path),
	}
}

func (p *Parser) parseCommodity() (syntax.Commodity, error) {
	r, err := p.ReadWhile1(isAlphanumeric)
	return syntax.Commodity(r), err
}

func (p *Parser) parseDecimal() (syntax.Decimal, error) {
	start := p.Offset()
	if _, err := p.ReadCharacterOpt('-'); err != nil {
		return syntax.Decimal(p.Range(start)), err
	}
	if _, err := p.ReadWhile1(unicode.IsDigit); err != nil {
		return syntax.Decimal(p.Range(start)), err
	}
	r, err := p.ReadCharacterOpt('.')
	if err != nil {
		return syntax.Decimal(p.Range(start)), err
	}
	if !r.Empty() {
		_, err = p.ReadWhile1(unicode.IsDigit)
	}
	return syntax.Decimal(p.Range(start)), err
}

func (p *Parser) parseAccount() (syntax.Account, error) {
	start := p.Offset()
	if _, err := p.ReadWhile1(isAlphanumeric); err != nil {
		return syntax.Account(p.Range(start)), err
	}
	for {
		if r, err := p.ReadCharacterOpt(':'); err != nil || r.Empty() {
			return syntax.Account(p.Range(start)), err
		}
		if _, err := p.ReadWhile1(isAlphanumeric); err != nil {
			return syntax.Account(p.Range(start)), err
		}
	}
}

func isAlphanumeric(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}
