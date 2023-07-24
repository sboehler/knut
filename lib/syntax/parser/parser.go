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
		Scanner: *scanner.New(text, path),
	}
}

func (p *Parser) parseCommodity() (syntax.Commodity, error) {
	r, err := p.ReadWhile1("a letter or a digit", isAlphanumeric)
	return syntax.Commodity{Range: r}, err
}

func (p *Parser) parseDecimal() (syntax.Decimal, error) {
	p.RangeStart()
	defer p.RangeEnd()
	if p.Current() == '-' {
		if _, err := p.ReadCharacter('-'); err != nil {
			return syntax.Decimal{Range: p.Range()}, err
		}
	}
	if _, err := p.ReadWhile1("a digit", unicode.IsDigit); err != nil {
		return syntax.Decimal{Range: p.Range()}, err
	}
	if p.Current() != '.' {
		return syntax.Decimal{Range: p.Range()}, nil
	}
	if _, err := p.ReadCharacter('.'); err != nil {
		return syntax.Decimal{Range: p.Range()}, err
	}
	_, err := p.ReadWhile1("a digit", unicode.IsDigit)
	return syntax.Decimal{Range: p.Range()}, err
}

func (p *Parser) parseAccount() (syntax.Account, error) {
	p.RangeStart()
	defer p.RangeEnd()
	if _, err := p.ReadWhile1("a letter or a digit", isAlphanumeric); err != nil {
		return syntax.Account{Range: p.Range()}, err
	}
	for {
		if p.Current() != ':' {
			return syntax.Account{Range: p.Range()}, nil
		}
		if _, err := p.ReadCharacter(':'); err != nil {
			return syntax.Account{Range: p.Range()}, err
		}
		if _, err := p.ReadWhile1("a letter or a digit", isAlphanumeric); err != nil {
			return syntax.Account{Range: p.Range()}, err
		}
	}
}

func (p *Parser) parseAccountMacro() (syntax.AccountMacro, error) {
	p.RangeStart()
	defer p.RangeEnd()
	if _, err := p.ReadCharacter('$'); err != nil {
		return syntax.AccountMacro{Range: p.Range()}, err
	}
	_, err := p.ReadWhile1("a letter", unicode.IsLetter)
	return syntax.AccountMacro{Range: p.Range()}, err
}

func (p *Parser) parseBooking() (syntax.Booking, error) {
	p.RangeStart()
	defer p.RangeEnd()
	booking := syntax.Booking{}
	var err error
	if p.Current() == '$' {
		if booking.CreditMacro, err = p.parseAccountMacro(); err != nil {
			return booking.SetRange(p.Range()), err
		}
	} else {
		if booking.Credit, err = p.parseAccount(); err != nil {
			return booking.SetRange(p.Range()), err
		}
	}
	if _, err := p.ReadWhile1("whitespace", isWhitespace); err != nil {
		return booking.SetRange(p.Range()), err
	}
	if p.Current() == '$' {
		if booking.DebitMacro, err = p.parseAccountMacro(); err != nil {
			return booking.SetRange(p.Range()), err
		}
	} else {
		if booking.Debit, err = p.parseAccount(); err != nil {
			return booking.SetRange(p.Range()), err
		}
	}
	if _, err := p.ReadWhile1("whitespace", isWhitespace); err != nil {
		return booking.SetRange(p.Range()), err
	}
	if booking.Amount, err = p.parseDecimal(); err != nil {
		return booking.SetRange(p.Range()), err
	}
	if _, err := p.ReadWhile1("whitespace", isWhitespace); err != nil {
		return booking.SetRange(p.Range()), err
	}
	booking.Commodity, err = p.parseCommodity()
	return booking.SetRange(p.Range()), err
}

func (p *Parser) parseDate() (syntax.Date, error) {
	p.RangeStart()
	defer p.RangeEnd()
	annotateError := func(desc string, err error) error {
		return syntax.Error{
			Message: desc,
			Range:   p.Range(),
			Wrapped: err,
		}
	}

	for i := 0; i < 4; i++ {
		if _, err := p.ReadCharacterWith("a digit", unicode.IsDigit); err != nil {
			return syntax.Date{Range: p.Range()}, annotateError("while parsing the date", err)
		}
	}
	for i := 0; i < 2; i++ {
		if _, err := p.ReadCharacter('-'); err != nil {
			return syntax.Date{Range: p.Range()}, annotateError("while parsing the date", err)
		}
		for j := 0; j < 2; j++ {
			if _, err := p.ReadCharacterWith("a digit", unicode.IsDigit); err != nil {
				return syntax.Date{Range: p.Range()}, annotateError("while parsing the date", err)
			}
		}
	}
	return syntax.Date{Range: p.Range()}, nil
}

func (p *Parser) parseQuotedString() (syntax.QuotedString, error) {
	p.RangeStart()
	defer p.RangeEnd()
	if _, err := p.ReadCharacter('"'); err != nil {
		return syntax.QuotedString{Range: p.Range()}, syntax.Error{
			Message: "while parsing quoted string",
			Range:   p.Range(),
			Wrapped: err,
		}
	}
	if _, err := p.ReadWhile(func(r rune) bool { return r != '"' }); err != nil {
		return syntax.QuotedString{Range: p.Range()}, syntax.Error{
			Message: "while parsing quoted string",
			Range:   p.Range(),
			Wrapped: err,
		}
	}
	if _, err := p.ReadCharacter('"'); err != nil {
		return syntax.QuotedString{Range: p.Range()}, syntax.Error{
			Message: "while parsing quoted string",
			Range:   p.Range(),
			Wrapped: err,
		}
	}
	return syntax.QuotedString{Range: p.Range()}, nil
}

func (p *Parser) parseTransaction(d syntax.Date, addons syntax.Addons) (syntax.Transaction, error) {
	p.RangeStart()
	defer p.RangeEnd()
	trx := syntax.Transaction{}
	var err error
	if trx.Description, err = p.parseQuotedString(); err != nil {
		return trx.SetRange(p.Range()), err
	}
	if _, err := p.readRestOfWhitespaceLine(); err != nil {
		return trx.SetRange(p.Range()), err
	}
	for {
		b, err := p.parseBooking()
		if err != nil {
			return trx.SetRange(p.Range()), err
		}
		trx.Bookings = append(trx.Bookings, b)
		if _, err := p.readRestOfWhitespaceLine(); err != nil {
			return trx.SetRange(p.Range()), err
		}
		if isWhitespaceOrNewline(p.Current()) || p.Current() == scanner.EOF {
			break
		}
	}
	return trx.SetRange(p.Range()), nil
}

func (p *Parser) readWhitespace1() (syntax.Range, error) {
	p.RangeStart()
	defer p.RangeEnd()
	if !isWhitespaceOrNewline(p.Current()) && p.Current() != scanner.EOF {
		return p.Range(), fmt.Errorf("expected whitespace, got %q", p.Current())
	}
	return p.ReadWhile(isWhitespace)
}

func (p *Parser) readRestOfWhitespaceLine() (syntax.Range, error) {
	p.RangeStart()
	defer p.RangeEnd()
	if _, err := p.ReadWhile(isWhitespace); err != nil {
		return p.Range(), err
	}
	_, err := p.ReadCharacter('\n')
	return p.Range(), err
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
