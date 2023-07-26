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
	p.RangeStart("parsing commodity")
	defer p.RangeEnd()
	r, err := p.ReadWhile1("a letter or a digit", isAlphanumeric)
	if err != nil {
		return syntax.Commodity{Range: p.Range()}, p.Annotate(err)
	}
	return syntax.Commodity{Range: r}, nil
}

func (p *Parser) parseDecimal() (syntax.Decimal, error) {
	p.RangeStart("parsing decimal")
	defer p.RangeEnd()
	if p.Current() == '-' {
		if _, err := p.ReadCharacter('-'); err != nil {
			return syntax.Decimal{Range: p.Range()}, p.Annotate(err)
		}
	}
	if _, err := p.ReadWhile1("a digit", unicode.IsDigit); err != nil {
		return syntax.Decimal{Range: p.Range()}, p.Annotate(err)
	}
	if p.Current() != '.' {
		return syntax.Decimal{Range: p.Range()}, nil
	}
	if _, err := p.ReadCharacter('.'); err != nil {
		return syntax.Decimal{Range: p.Range()}, p.Annotate(err)
	}
	if _, err := p.ReadWhile1("a digit", unicode.IsDigit); err != nil {
		return syntax.Decimal{Range: p.Range()}, p.Annotate(err)
	}
	return syntax.Decimal{Range: p.Range()}, nil
}

func (p *Parser) parseAccount() (syntax.Account, error) {
	p.RangeStart("parsing account")
	defer p.RangeEnd()
	if _, err := p.ReadWhile1("a letter or a digit", isAlphanumeric); err != nil {
		return syntax.Account{Range: p.Range()}, p.Annotate(err)
	}
	for {
		if p.Current() != ':' {
			return syntax.Account{Range: p.Range()}, nil
		}
		if _, err := p.ReadCharacter(':'); err != nil {
			return syntax.Account{Range: p.Range()}, p.Annotate(err)
		}
		if _, err := p.ReadWhile1("a letter or a digit", isAlphanumeric); err != nil {
			return syntax.Account{Range: p.Range()}, p.Annotate(err)
		}
	}
}

func (p *Parser) parseAccountMacro() (syntax.AccountMacro, error) {
	p.RangeStart("parsing account macro")
	defer p.RangeEnd()
	if _, err := p.ReadCharacter('$'); err != nil {
		return syntax.AccountMacro{Range: p.Range()}, p.Annotate(err)
	}
	if _, err := p.ReadWhile1("a letter", unicode.IsLetter); err != nil {
		return syntax.AccountMacro{Range: p.Range()}, p.Annotate(err)
	}
	return syntax.AccountMacro{Range: p.Range()}, nil
}

func (p *Parser) parseBooking() (syntax.Booking, error) {
	p.RangeStart("parsing booking")
	defer p.RangeEnd()
	booking := syntax.Booking{}
	var err error
	if p.Current() == '$' {
		if booking.CreditMacro, err = p.parseAccountMacro(); err != nil {
			return booking.SetRange(p.Range()), p.Annotate(err)
		}
	} else {
		if booking.Credit, err = p.parseAccount(); err != nil {
			return booking.SetRange(p.Range()), p.Annotate(err)
		}
	}
	if _, err := p.ReadWhile1("whitespace", isWhitespace); err != nil {
		return booking.SetRange(p.Range()), p.Annotate(err)
	}
	if p.Current() == '$' {
		if booking.DebitMacro, err = p.parseAccountMacro(); err != nil {
			return booking.SetRange(p.Range()), p.Annotate(err)
		}
	} else {
		if booking.Debit, err = p.parseAccount(); err != nil {
			return booking.SetRange(p.Range()), p.Annotate(err)
		}
	}
	if _, err := p.ReadWhile1("whitespace", isWhitespace); err != nil {
		return booking.SetRange(p.Range()), p.Annotate(err)
	}
	if booking.Amount, err = p.parseDecimal(); err != nil {
		return booking.SetRange(p.Range()), p.Annotate(err)
	}
	if _, err := p.ReadWhile1("whitespace", isWhitespace); err != nil {
		return booking.SetRange(p.Range()), p.Annotate(err)
	}
	if booking.Commodity, err = p.parseCommodity(); err != nil {
		return booking.SetRange(p.Range()), err
	}
	return booking.SetRange(p.Range()), nil
}

func (p *Parser) parseDate() (syntax.Date, error) {
	p.RangeStart("parsing the date")
	defer p.RangeEnd()

	for i := 0; i < 4; i++ {
		if _, err := p.ReadCharacterWith("a digit", unicode.IsDigit); err != nil {
			return syntax.Date{Range: p.Range()}, p.Annotate(err)
		}
	}
	for i := 0; i < 2; i++ {
		if _, err := p.ReadCharacter('-'); err != nil {
			return syntax.Date{Range: p.Range()}, p.Annotate(err)
		}
		for j := 0; j < 2; j++ {
			if _, err := p.ReadCharacterWith("a digit", unicode.IsDigit); err != nil {
				return syntax.Date{Range: p.Range()}, p.Annotate(err)
			}
		}
	}
	return syntax.Date{Range: p.Range()}, nil
}

func (p *Parser) parseQuotedString() (syntax.QuotedString, error) {
	p.RangeStart("parsing quoted string")
	defer p.RangeEnd()
	if _, err := p.ReadCharacter('"'); err != nil {
		return syntax.QuotedString{Range: p.Range()}, p.Annotate(err)
	}
	if _, err := p.ReadWhile(func(r rune) bool { return r != '"' }); err != nil {
		return syntax.QuotedString{Range: p.Range()}, p.Annotate(err)
	}
	if _, err := p.ReadCharacter('"'); err != nil {
		return syntax.QuotedString{Range: p.Range()}, p.Annotate(err)
	}
	return syntax.QuotedString{Range: p.Range()}, nil
}

func (p *Parser) parseTransaction(d syntax.Date, addons syntax.Addons) (syntax.Transaction, error) {
	p.RangeStart("parsing transaction")
	defer p.RangeEnd()
	trx := syntax.Transaction{}
	var err error
	if trx.Description, err = p.parseQuotedString(); err != nil {
		return trx.SetRange(p.Range()), p.Annotate(err)
	}
	if _, err := p.readRestOfWhitespaceLine(); err != nil {
		return trx.SetRange(p.Range()), p.Annotate(err)
	}
	for {
		b, err := p.parseBooking()
		trx.Bookings = append(trx.Bookings, b)
		if err != nil {
			return trx.SetRange(p.Range()), p.Annotate(err)
		}
		if _, err := p.readRestOfWhitespaceLine(); err != nil {
			return trx.SetRange(p.Range()), p.Annotate(err)
		}
		if isWhitespaceOrNewline(p.Current()) || p.Current() == scanner.EOF {
			break
		}
	}
	return trx.SetRange(p.Range()), nil
}

func (p *Parser) parseAddons() (syntax.Addons, error) {
	p.RangeStart("parsing addons")
	defer p.RangeEnd()
	addons := syntax.Addons{}
	for {
		r, err := p.ReadAlternative([]string{"@performance", "@accrue"})
		if err != nil {
			return addons.SetRange(r), p.Annotate(err)
		}
		switch r.Extract() {
		case "@performance":
			if !addons.Performance.Empty() {
				return addons.SetRange(p.Range()), p.Annotate(syntax.Error{
					Message: "duplicate @performance annotation",
					Range:   r,
				})
			}
			addons.Performance, err = p.parsePerformance()
			addons.Performance.Extend(r)
			if err != nil {
				return addons.SetRange(p.Range()), p.Annotate(err)
			}

		case "@accrue":
			if !addons.Accrual.Empty() {
				return addons.SetRange(p.Range()), p.Annotate(syntax.Error{
					Message: "duplicate @performance annotation",
					Range:   r,
				})
			}
			addons.Accrual, err = p.parseAccrual()
			addons.Accrual.Extend(r)
			if err != nil {
				return addons.SetRange(p.Range()), p.Annotate(err)
			}
		}
		if _, err := p.readRestOfWhitespaceLine(); err != nil {
			return addons.SetRange(p.Range()), p.Annotate(syntax.Error{})
		}
		if p.Current() != '@' {
			return addons.SetRange(p.Range()), nil
		}
	}
}

func (p *Parser) parsePerformance() (syntax.Performance, error) {
	p.RangeStart("parsing performance")
	defer p.RangeEnd()
	perf := syntax.Performance{Range: p.Range()}
	if _, err := p.ReadCharacter('('); err != nil {
		return perf.SetRange(p.Range()), p.Annotate(err)
	}
	if _, err := p.ReadWhile(isWhitespace); err != nil {
		return perf.SetRange(p.Range()), p.Annotate(err)
	}
	if p.Current() != ')' {
		if c, err := p.parseCommodity(); err != nil {
			return perf.SetRange(p.Range()), p.Annotate(err)
		} else {
			perf.Targets = append(perf.Targets, c)
		}
		if _, err := p.ReadWhile(isWhitespace); err != nil {
			return perf.SetRange(p.Range()), p.Annotate(err)
		}
	}
	for p.Current() == ',' {
		if _, err := p.ReadCharacter(','); err != nil {
			return perf.SetRange(p.Range()), p.Annotate(err)
		}
		if _, err := p.ReadWhile(isWhitespace); err != nil {
			return perf.SetRange(p.Range()), p.Annotate(err)
		}
		if c, err := p.parseCommodity(); err != nil {
			return perf.SetRange(p.Range()), p.Annotate(err)
		} else {
			perf.Targets = append(perf.Targets, c)
		}
		if _, err := p.ReadWhile(isWhitespace); err != nil {
			return perf.SetRange(p.Range()), p.Annotate(err)
		}
	}
	if _, err := p.ReadCharacter(')'); err != nil {
		return perf.SetRange(p.Range()), p.Annotate(err)
	}
	return perf.SetRange(p.Range()), nil
}

func (p *Parser) parseAccrual() (syntax.Accrual, error) {
	p.RangeStart("parsing addons")
	defer p.RangeEnd()
	accrual := syntax.Accrual{Range: p.Range()}
	if _, err := p.readWhitespace1(); err != nil {
		return accrual.SetRange(p.Range()), p.Annotate(err)
	}
	var err error
	if accrual.Interval, err = p.parseInterval(); err != nil {
		return accrual.SetRange(p.Range()), p.Annotate(err)
	}
	if _, err := p.readWhitespace1(); err != nil {
		return accrual.SetRange(p.Range()), p.Annotate(err)
	}
	if accrual.Start, err = p.parseDate(); err != nil {
		return accrual.SetRange(p.Range()), p.Annotate(err)
	}
	if _, err := p.readWhitespace1(); err != nil {
		return accrual.SetRange(p.Range()), p.Annotate(err)
	}
	if accrual.End, err = p.parseDate(); err != nil {
		return accrual.SetRange(p.Range()), p.Annotate(err)
	}
	if _, err := p.readWhitespace1(); err != nil {
		return accrual.SetRange(p.Range()), p.Annotate(err)
	}
	if accrual.Account, err = p.parseAccount(); err != nil {
		return accrual.SetRange(p.Range()), p.Annotate(err)
	}
	return accrual.SetRange(p.Range()), nil
}

func (p *Parser) parseInterval() (syntax.Interval, error) {
	p.RangeStart("parsing interval")
	defer p.RangeEnd()
	if _, err := p.ReadAlternative([]string{"daily", "weekly", "monthly", "quarterly"}); err != nil {
		return syntax.Interval{Range: p.Range()}, p.Annotate(err)
	}
	return syntax.Interval{Range: p.Range()}, nil
}

func (p *Parser) readWhitespace1() (syntax.Range, error) {
	p.RangeStart("")
	defer p.RangeEnd()
	if !isWhitespaceOrNewline(p.Current()) && p.Current() != scanner.EOF {
		return p.Range(), syntax.Error{
			Message: fmt.Sprintf("unexpected character `%c`, want whitespace or a newline", p.Current()),
			Range:   p.Range(),
		}
	}
	return p.ReadWhile(isWhitespace)
}

func (p *Parser) readRestOfWhitespaceLine() (syntax.Range, error) {
	p.RangeStart("")
	defer p.RangeEnd()
	if _, err := p.ReadWhile(isWhitespace); err != nil {
		return p.Range(), err
	}
	if p.Current() == scanner.EOF {
		return p.Range(), nil
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
