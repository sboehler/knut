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
	r, err := p.parseIdentifier()
	return syntax.Commodity(r), err
}

func (p *Parser) parseIdentifier() (syntax.Pos, error) {
	return p.ReadWhile1(isAlphanumeric)
}

func isAlphanumeric(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}
