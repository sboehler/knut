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
	return syntax.Commodity{Range: r}, err
}

func (p *Parser) parseDecimal() (syntax.Decimal, error) {
	decimal := syntax.Decimal{Range: p.Range()}
	if p.Current() == '-' {
		if _, err := p.ReadCharacter('-'); err != nil {
			return updateRange(p, &decimal), err
		}
	}
	if _, err := p.ReadWhile1(unicode.IsDigit); err != nil {
		return updateRange(p, &decimal), err
	}
	if p.Current() != '.' {
		return updateRange(p, &decimal), nil
	}
	if _, err := p.ReadCharacter('.'); err != nil {
		return updateRange(p, &decimal), err
	}
	_, err := p.ReadWhile1(unicode.IsDigit)
	return updateRange(p, &decimal), err
}

func (p *Parser) parseAccount() (syntax.Account, error) {
	account := syntax.Account{Range: p.Range()}
	if _, err := p.ReadWhile1(isAlphanumeric); err != nil {
		return updateRange(p, &account), err
	}
	for {
		if p.Current() != ':' {
			return updateRange(p, &account), nil
		}
		if _, err := p.ReadCharacter(':'); err != nil {
			return updateRange(p, &account), err
		}
		if _, err := p.ReadWhile1(isAlphanumeric); err != nil {
			return updateRange(p, &account), err
		}
	}
}

func (p *Parser) parseAccountMacro() (syntax.AccountMacro, error) {
	macro := syntax.AccountMacro{Range: p.Range()}
	if _, err := p.ReadCharacter('$'); err != nil {
		return updateRange(p, &macro), err
	}
	_, err := p.ReadWhile1(unicode.IsLetter)
	return updateRange(p, &macro), err
}

func (p *Parser) parseBooking() (syntax.Booking, error) {
	booking := syntax.Booking{Range: p.Range()}
	var err error
	if p.Current() == '$' {
		if booking.CreditMacro, err = p.parseAccountMacro(); err != nil {
			return updateRange(p, &booking), err
		}
	} else {
		if booking.Credit, err = p.parseAccount(); err != nil {
			return updateRange(p, &booking), err
		}
	}
	if _, err := p.ReadWhile1(isWhitespace); err != nil {
		return updateRange(p, &booking), err
	}
	if p.Current() == '$' {
		if booking.DebitMacro, err = p.parseAccountMacro(); err != nil {
			return updateRange(p, &booking), err
		}
	} else {
		if booking.Debit, err = p.parseAccount(); err != nil {
			return updateRange(p, &booking), err
		}
	}
	if _, err := p.ReadWhile1(isWhitespace); err != nil {
		return updateRange(p, &booking), err
	}
	if booking.Amount, err = p.parseDecimal(); err != nil {
		return updateRange(p, &booking), err
	}
	if _, err := p.ReadWhile1(isWhitespace); err != nil {
		return updateRange(p, &booking), err
	}
	booking.Commodity, err = p.parseCommodity()
	return updateRange(p, &booking), err
}

func updateRange[P interface {
	*T
	SetEnd(int)
}, T any](p *Parser, b P) T {
	b.SetEnd(p.Offset())
	return *b
}

func (p *Parser) parseDate() (syntax.Date, error) {
	date := syntax.Date{Range: p.Range()}
	for i := 0; i < 4; i++ {
		if _, err := p.ReadCharacterWith(unicode.IsDigit); err != nil {
			return updateRange(p, &date), err
		}
	}
	for i := 0; i < 2; i++ {
		if _, err := p.ReadCharacter('-'); err != nil {
			return updateRange(p, &date), err
		}
		for j := 0; j < 2; j++ {
			if _, err := p.ReadCharacterWith(unicode.IsDigit); err != nil {
				return updateRange(p, &date), err
			}
		}
	}
	return updateRange(p, &date), nil
}

func (p *Parser) parseQuotedString() (syntax.QuotedString, error) {
	qs := syntax.QuotedString{Range: p.Range()}
	if _, err := p.ReadCharacter('"'); err != nil {
		return updateRange(p, &qs), err
	}
	if _, err := p.ReadWhile(func(r rune) bool { return r != '"' }); err != nil {
		return updateRange(p, &qs), err
	}
	_, err := p.ReadCharacter('"')
	return updateRange(p, &qs), err
}

func (p *Parser) parseTransaction(d syntax.Date, addons syntax.Addons) (syntax.Transaction, error) {
	trx := syntax.Transaction{Range: p.Range()}
	var err error
	if trx.Description, err = p.parseQuotedString(); err != nil {
		return updateRange(p, &trx), err
	}
	if _, err := p.readRestOfWhitespaceLine(); err != nil {
		return updateRange(p, &trx), err
	}
	for {
		b, err := p.parseBooking()
		if err != nil {
			return updateRange(p, &trx), err
		}
		trx.Bookings = append(trx.Bookings, b)
		if _, err := p.readRestOfWhitespaceLine(); err != nil {
			return updateRange(p, &trx), err
		}
		if isWhitespaceOrNewline(p.Current()) || p.Current() == scanner.EOF {
			break
		}
	}
	return updateRange(p, &trx), nil
}

func (p *Parser) readWhitespace1() (syntax.Range, error) {
	if !isWhitespaceOrNewline(p.Current()) && p.Current() != scanner.EOF {
		return p.Range(), fmt.Errorf("expected whitespace, got %q", p.Current())
	}
	return p.ReadWhile(isWhitespace)
}

func (p *Parser) readRestOfWhitespaceLine() (syntax.Range, error) {
	rng := p.Range()
	if _, err := p.ReadWhile(isWhitespace); err != nil {
		return updateRange(p, &rng), err
	}
	_, err := p.ReadCharacter('\n')
	return updateRange(p, &rng), err
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
