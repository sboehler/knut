package parser

import (
	"fmt"
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
	if p.Current() == '-' {
		if _, err := p.ReadCharacter('-'); err != nil {
			return syntax.Decimal(p.Range(start)), err
		}
	}
	if _, err := p.ReadWhile1(unicode.IsDigit); err != nil {
		return syntax.Decimal(p.Range(start)), err
	}
	if p.Current() != '.' {
		return syntax.Decimal(p.Range(start)), nil
	}
	if _, err := p.ReadCharacter('.'); err != nil {
		return syntax.Decimal(p.Range(start)), err
	}
	_, err := p.ReadWhile1(unicode.IsDigit)
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

func (p *Parser) parseDate() (syntax.Date, error) {
	start := p.Offset()
	for i := 0; i < 4; i++ {
		if _, err := p.ReadCharacterWith(unicode.IsDigit); err != nil {
			return syntax.Date(p.Range(start)), err
		}
	}
	for i := 0; i < 2; i++ {
		if _, err := p.ReadCharacter('-'); err != nil {
			return syntax.Date(p.Range(start)), err
		}
		for j := 0; j < 2; j++ {
			if _, err := p.ReadCharacterWith(unicode.IsDigit); err != nil {
				return syntax.Date(p.Range(start)), err
			}
		}
	}
	return syntax.Date(p.Range(start)), nil
}

func (p *Parser) parseQuotedString() (syntax.QuotedString, error) {
	start := p.Offset()
	if _, err := p.ReadCharacter('"'); err != nil {
		return syntax.QuotedString(p.Range(start)), err
	}
	if _, err := p.ReadWhile(func(r rune) bool { return r != '"' }); err != nil {
		return syntax.QuotedString(p.Range(start)), err
	}
	_, err := p.ReadCharacter('"')
	return syntax.QuotedString(p.Range(start)), err
}

func (p *Parser) parseTransaction(d syntax.Date, addons syntax.Addons) (syntax.Transaction, error) {
	trx := syntax.Transaction{Pos: p.Rng()}
	var err error
	if trx.Description, err = p.parseQuotedString(); err != nil {
		return p.finishTransaction(trx), err
	}
	if _, err := p.readRestOfWhitespaceLine(); err != nil {
		return p.finishTransaction(trx), err
	}
	for {
		b, err := p.parseBooking()
		if err != nil {
			return p.finishTransaction(trx), err
		}
		trx.Bookings = append(trx.Bookings, b)
		if _, err := p.readRestOfWhitespaceLine(); err != nil {
			return p.finishTransaction(trx), err
		}
		if isWhitespaceOrNewline(p.Current()) || p.Current() == scanner.EOF {
			break
		}
	}
	return p.finishTransaction(trx), nil
}

func (p *Parser) finishTransaction(b syntax.Transaction) syntax.Transaction {
	b.End = p.Offset()
	return b
}

func (p *Parser) readWhitespace1() (syntax.Pos, error) {
	if !isWhitespaceOrNewline(p.Current()) && p.Current() != scanner.EOF {
		return p.Rng(), fmt.Errorf("expected whitespace, got %q", p.Current())
	}
	return p.ReadWhile(isWhitespace)
}

func (p *Parser) readRestOfWhitespaceLine() (syntax.Pos, error) {
	start := p.Offset()
	if _, err := p.ReadWhile(isWhitespace); err != nil {
		return p.Range(start), err
	}
	_, err := p.ReadCharacter('\n')
	return p.Range(start), err
}

func isAlphanumeric(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

func isWhitespace(ch rune) bool {
	return ch == ' ' || ch == '\t' || ch == '\r'
}
func isWhitespaceOrNewline(ch rune) bool {
	return isNewline(ch) || isWhitespace(ch)
}

func isNewline(ch rune) bool {
	return ch == '\n'
}
