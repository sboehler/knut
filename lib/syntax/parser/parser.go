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
		if p.Current() != ':' {
			return syntax.Account(p.Range(start)), nil
		}
		if _, err := p.ReadCharacter(':'); err != nil {
			return syntax.Account(p.Range(start)), err
		}
		if _, err := p.ReadWhile1(isAlphanumeric); err != nil {
			return syntax.Account(p.Range(start)), err
		}
	}
}

func (p *Parser) parseAccountMacro() (syntax.AccountMacro, error) {
	start := p.Offset()
	if _, err := p.ReadCharacter('$'); err != nil {
		return syntax.AccountMacro(p.Range(start)), err
	}
	_, err := p.ReadWhile1(unicode.IsLetter)
	return syntax.AccountMacro(p.Range(start)), err
}

func (p *Parser) parseBooking() (syntax.Booking, error) {
	booking := syntax.Booking{Pos: p.Rng()}
	var err error
	if p.Current() == '$' {
		if booking.CreditMacro, err = p.parseAccountMacro(); err != nil {
			return p.finishBooking(booking), err
		}
	} else {
		if booking.Credit, err = p.parseAccount(); err != nil {
			return p.finishBooking(booking), err
		}
	}
	if _, err := p.ReadWhile1(isWhitespace); err != nil {
		return p.finishBooking(booking), err
	}
	if p.Current() == '$' {
		if booking.DebitMacro, err = p.parseAccountMacro(); err != nil {
			return p.finishBooking(booking), err
		}
	} else {
		if booking.Debit, err = p.parseAccount(); err != nil {
			return p.finishBooking(booking), err
		}
	}
	if _, err := p.ReadWhile1(isWhitespace); err != nil {
		return p.finishBooking(booking), err
	}
	if booking.Amount, err = p.parseDecimal(); err != nil {
		return p.finishBooking(booking), err
	}
	if _, err := p.ReadWhile1(isWhitespace); err != nil {
		return p.finishBooking(booking), err
	}
	booking.Commodity, err = p.parseCommodity()
	return p.finishBooking(booking), err
}

func (p *Parser) finishBooking(b syntax.Booking) syntax.Booking {
	b.End = p.Offset()
	return b
}

func isAlphanumeric(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

func isWhitespace(ch rune) bool {
	return ch == ' ' || ch == '\t' || ch == '\r'
}
