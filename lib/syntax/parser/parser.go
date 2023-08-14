package parser

import (
	"fmt"
	"unicode"

	"github.com/sboehler/knut/lib/syntax/directives"
	"github.com/sboehler/knut/lib/syntax/scanner"
)

// Parser parses a journal.
type Parser struct {
	scanner.Scanner

	Callback func(d directives.Directive)
}

// New creates a new parser.
func New(text, path string) *Parser {
	return &Parser{
		Scanner: *scanner.New(text, path),
	}
}

func (p *Parser) readComment() (directives.Range, error) {
	p.RangeStart("reading comment")
	defer p.RangeEnd()
	if _, err := p.ReadAlternative([]string{"*", "//", "#"}); err != nil {
		return p.Range(), p.Annotate(err)
	}
	if _, err := p.ReadWhile(func(r rune) bool { return !isNewlineOrEOF(r) }); err != nil {
		return p.Range(), p.Annotate(err)
	}
	return p.Range(), nil
}

func (p *Parser) ParseFile() (directives.File, error) {
	p.RangeStart(fmt.Sprintf("parsing file `%s`", p.Path))
	defer p.RangeEnd()
	var file directives.File
	for p.Current() != scanner.EOF {
		switch {

		case p.Current() == '*' || p.Current() == '#' || p.Current() == '/':
			if _, err := p.readComment(); err != nil {
				return directives.SetRange(&file, p.Range()), p.Annotate(err)
			}

		case isAlphanumeric(p.Current()) || p.Current() == '@':
			dir, err := p.parseDirective()
			file.Directives = append(file.Directives, dir)
			if err != nil {
				return directives.SetRange(&file, p.Range()), p.Annotate(err)
			}
			if p.Callback != nil {
				p.Callback(dir)
			}
		}
		if p.Current() == scanner.EOF {
			break
		}
		if _, err := p.readRestOfWhitespaceLine(); err != nil {
			return directives.SetRange(&file, p.Range()), p.Annotate(err)
		}
	}
	return directives.SetRange(&file, p.Range()), nil
}

func (p *Parser) parseDirective() (directives.Directive, error) {
	p.RangeStart("parsing directive")
	defer p.RangeEnd()
	var (
		dir    directives.Directive
		addons directives.Addons
	)
	var err error
	if p.Current() == '@' {
		if addons, err = p.parseAddons(); err != nil {
			return directives.SetRange(&dir, p.Range()), p.Annotate(err)
		}
	}
	if p.Current() == 'i' {
		if dir.Directive, err = p.parseInclude(); err != nil {
			return directives.SetRange(&dir, p.Range()), p.Annotate(err)
		}
	} else {
		date, err := p.parseDate()
		if err != nil {
			return directives.SetRange(&dir, p.Range()), p.Annotate(err)
		}
		if _, err := p.readWhitespace1(); err != nil {
			return directives.SetRange(&dir, p.Range()), p.Annotate(err)
		}
		if p.Current() == '"' {
			if dir.Directive, err = p.parseTransaction(date, addons); err != nil {
				return directives.SetRange(&dir, p.Range()), p.Annotate(err)
			}
		} else {
			r, err := p.ReadAlternative([]string{"open", "close", "balance", "price"})
			if err != nil {
				return directives.SetRange(&dir, p.Range()), p.Annotate(err)
			}
			if _, err := p.readWhitespace1(); err != nil {
				return directives.SetRange(&dir, p.Range()), p.Annotate(err)
			}
			switch r.Extract() {
			case "open":
				if dir.Directive, err = p.parseOpen(date); err != nil {
					return directives.SetRange(&dir, p.Range()), p.Annotate(err)
				}
			case "close":
				if dir.Directive, err = p.parseClose(date); err != nil {
					return directives.SetRange(&dir, p.Range()), p.Annotate(err)
				}
			case "balance":
				if dir.Directive, err = p.parseAssertion(date); err != nil {
					return directives.SetRange(&dir, p.Range()), p.Annotate(err)
				}
			case "price":
				if dir.Directive, err = p.parsePrice(date); err != nil {
					return directives.SetRange(&dir, p.Range()), p.Annotate(err)
				}
			}
		}
	}
	return directives.SetRange(&dir, p.Range()), nil
}

func (p *Parser) parseInclude() (directives.Include, error) {
	p.RangeStart("parsing `include` statement")
	defer p.RangeEnd()
	var (
		include = directives.Include{}
		err     error
	)
	if _, err := p.ReadString("include"); err != nil {
		return directives.SetRange(&include, p.Range()), p.Annotate(err)
	}
	if _, err := p.readWhitespace1(); err != nil {
		return directives.SetRange(&include, p.Range()), p.Annotate(err)
	}
	if include.IncludePath, err = p.parseQuotedString(); err != nil {
		return directives.SetRange(&include, p.Range()), p.Annotate(err)
	}
	return directives.SetRange(&include, p.Range()), nil
}

func (p *Parser) parseOpen(date directives.Date) (directives.Open, error) {
	p.RangeContinue("parsing `open` directive")
	defer p.RangeEnd()
	var (
		open = directives.Open{Date: date}
		err  error
	)
	if open.Account, err = p.parseAccount(); err != nil {
		err = p.Annotate(err)
	}
	return directives.SetRange(&open, p.Range()), err
}

func (p *Parser) parseClose(date directives.Date) (directives.Close, error) {
	p.RangeContinue("parsing `close` directive")
	defer p.RangeEnd()
	var (
		close = directives.Close{Date: date}
		err   error
	)
	if close.Account, err = p.parseAccount(); err != nil {
		err = p.Annotate(err)
	}
	return directives.SetRange(&close, p.Range()), err
}

func (p *Parser) parseAssertion(date directives.Date) (directives.Assertion, error) {
	p.RangeContinue("parsing `balance` directive")
	defer p.RangeEnd()
	var (
		assertion = directives.Assertion{Date: date}
		err       error
	)
	if assertion.Account, err = p.parseAccount(); err != nil {
		return directives.SetRange(&assertion, p.Range()), p.Annotate(err)
	}
	if _, err := p.readWhitespace1(); err != nil {
		return directives.SetRange(&assertion, p.Range()), p.Annotate(err)
	}
	if assertion.Quantity, err = p.parseDecimal(); err != nil {
		return directives.SetRange(&assertion, p.Range()), p.Annotate(err)
	}
	if _, err := p.readWhitespace1(); err != nil {
		return directives.SetRange(&assertion, p.Range()), p.Annotate(err)
	}
	if assertion.Commodity, err = p.parseCommodity(); err != nil {
		err = p.Annotate(err)
	}
	return directives.SetRange(&assertion, p.Range()), err
}

func (p *Parser) parsePrice(date directives.Date) (directives.Price, error) {
	p.RangeContinue("parsing `balance` directive")
	defer p.RangeEnd()
	var (
		price = directives.Price{Date: date}
		err   error
	)
	if price.Commodity, err = p.parseCommodity(); err != nil {
		return directives.SetRange(&price, p.Range()), p.Annotate(err)
	}
	if _, err := p.readWhitespace1(); err != nil {
		return directives.SetRange(&price, p.Range()), p.Annotate(err)
	}
	if price.Price, err = p.parseDecimal(); err != nil {
		return directives.SetRange(&price, p.Range()), p.Annotate(err)
	}
	if _, err := p.readWhitespace1(); err != nil {
		return directives.SetRange(&price, p.Range()), p.Annotate(err)
	}
	if price.Target, err = p.parseCommodity(); err != nil {
		return directives.SetRange(&price, p.Range()), err
	}
	return directives.SetRange(&price, p.Range()), err
}

func (p *Parser) parseCommodity() (directives.Commodity, error) {
	var (
		commodity directives.Commodity
		err       error
	)
	p.RangeStart("parsing commodity")
	defer p.RangeEnd()
	_, err = p.ReadWhile1("a letter or a digit", isAlphanumeric)
	if err != nil {
		err = p.Annotate(err)
	}
	return directives.SetRange(&commodity, p.Range()), err
}

func (p *Parser) parseDecimal() (directives.Decimal, error) {
	p.RangeStart("parsing decimal")
	defer p.RangeEnd()
	if p.Current() == '-' {
		if _, err := p.ReadCharacter('-'); err != nil {
			return directives.Decimal{Range: p.Range()}, p.Annotate(err)
		}
	}
	if _, err := p.ReadWhile1("a digit", unicode.IsDigit); err != nil {
		return directives.Decimal{Range: p.Range()}, p.Annotate(err)
	}
	if p.Current() != '.' {
		return directives.Decimal{Range: p.Range()}, nil
	}
	if _, err := p.ReadCharacter('.'); err != nil {
		return directives.Decimal{Range: p.Range()}, p.Annotate(err)
	}
	if _, err := p.ReadWhile1("a digit", unicode.IsDigit); err != nil {
		return directives.Decimal{Range: p.Range()}, p.Annotate(err)
	}
	return directives.Decimal{Range: p.Range()}, nil
}

func (p *Parser) parseAccount() (directives.Account, error) {
	p.RangeStart("parsing account")
	defer p.RangeEnd()
	acc := directives.Account{}
	if p.Current() == '$' {
		acc.Macro = true
		if _, err := p.ReadCharacter('$'); err != nil {
			return directives.SetRange(&acc, p.Range()), p.Annotate(err)
		}
		if _, err := p.ReadWhile1("a letter", unicode.IsLetter); err != nil {
			return directives.SetRange(&acc, p.Range()), p.Annotate(err)
		}
		return directives.SetRange(&acc, p.Range()), nil
	}
	if _, err := p.ReadWhile1("a letter or a digit", isAlphanumeric); err != nil {
		return directives.Account{Range: p.Range()}, p.Annotate(err)
	}
	for {
		if p.Current() != ':' {
			return directives.Account{Range: p.Range()}, nil
		}
		if _, err := p.ReadCharacter(':'); err != nil {
			return directives.Account{Range: p.Range()}, p.Annotate(err)
		}
		if _, err := p.ReadWhile1("a letter or a digit", isAlphanumeric); err != nil {
			return directives.Account{Range: p.Range()}, p.Annotate(err)
		}
	}
}

func (p *Parser) parseBooking() (directives.Booking, error) {
	p.RangeStart("parsing booking")
	defer p.RangeEnd()
	var (
		booking directives.Booking
		err     error
	)
	if booking.Credit, err = p.parseAccount(); err != nil {
		return directives.SetRange(&booking, p.Range()), p.Annotate(err)
	}
	if _, err := p.ReadWhile1("whitespace", isWhitespace); err != nil {
		return directives.SetRange(&booking, p.Range()), p.Annotate(err)
	}
	if booking.Debit, err = p.parseAccount(); err != nil {
		return directives.SetRange(&booking, p.Range()), p.Annotate(err)
	}
	if _, err := p.ReadWhile1("whitespace", isWhitespace); err != nil {
		return directives.SetRange(&booking, p.Range()), p.Annotate(err)
	}
	if booking.Quantity, err = p.parseDecimal(); err != nil {
		return directives.SetRange(&booking, p.Range()), p.Annotate(err)
	}
	if _, err := p.ReadWhile1("whitespace", isWhitespace); err != nil {
		return directives.SetRange(&booking, p.Range()), p.Annotate(err)
	}
	if booking.Commodity, err = p.parseCommodity(); err != nil {
		return directives.SetRange(&booking, p.Range()), p.Annotate(err)
	}
	return directives.SetRange(&booking, p.Range()), nil
}

func (p *Parser) parseDate() (directives.Date, error) {
	p.RangeStart("parsing the date")
	defer p.RangeEnd()

	for i := 0; i < 4; i++ {
		if _, err := p.ReadCharacterWith("a digit", unicode.IsDigit); err != nil {
			return directives.Date{Range: p.Range()}, p.Annotate(err)
		}
	}
	for i := 0; i < 2; i++ {
		if _, err := p.ReadCharacter('-'); err != nil {
			return directives.Date{Range: p.Range()}, p.Annotate(err)
		}
		for j := 0; j < 2; j++ {
			if _, err := p.ReadCharacterWith("a digit", unicode.IsDigit); err != nil {
				return directives.Date{Range: p.Range()}, p.Annotate(err)
			}
		}
	}
	return directives.Date{Range: p.Range()}, nil
}

func (p *Parser) parseQuotedString() (directives.QuotedString, error) {
	p.RangeStart("parsing quoted string")
	defer p.RangeEnd()
	var (
		qs  directives.QuotedString
		err error
	)
	if _, err := p.ReadCharacter('"'); err != nil {
		return directives.SetRange(&qs, p.Range()), p.Annotate(err)
	}
	if qs.Content, err = p.ReadWhile(func(r rune) bool { return r != '"' }); err != nil {
		return directives.SetRange(&qs, p.Range()), p.Annotate(err)
	}
	if _, err := p.ReadCharacter('"'); err != nil {
		return directives.SetRange(&qs, p.Range()), p.Annotate(err)
	}
	return directives.SetRange(&qs, p.Range()), nil
}

func (p *Parser) parseTransaction(date directives.Date, addons directives.Addons) (directives.Transaction, error) {
	p.RangeContinue("parsing transaction")
	defer p.RangeEnd()
	var (
		trx = directives.Transaction{Date: date, Addons: addons}
		err error
	)
	if trx.Description, err = p.parseQuotedString(); err != nil {
		return directives.SetRange(&trx, p.Range()), p.Annotate(err)
	}
	if _, err := p.readRestOfWhitespaceLine(); err != nil {
		return directives.SetRange(&trx, p.Range()), p.Annotate(err)
	}
	for {
		b, err := p.parseBooking()
		trx.Bookings = append(trx.Bookings, b)
		if err != nil {
			return directives.SetRange(&trx, p.Range()), p.Annotate(err)
		}
		if _, err := p.readRestOfWhitespaceLine(); err != nil {
			return directives.SetRange(&trx, p.Range()), p.Annotate(err)
		}
		if isWhitespaceOrNewline(p.Current()) || p.Current() == scanner.EOF {
			break
		}
	}
	return directives.SetRange(&trx, p.Range()), nil
}

func (p *Parser) parseAddons() (directives.Addons, error) {
	p.RangeStart("parsing addons")
	defer p.RangeEnd()
	var addons directives.Addons
	for {
		r, err := p.ReadAlternative([]string{"@performance", "@accrue"})
		if err != nil {
			return directives.SetRange(&addons, r), p.Annotate(err)
		}
		switch r.Extract() {
		case "@performance":
			if !addons.Performance.Empty() {
				return directives.SetRange(&addons, p.Range()), p.Annotate(directives.Error{
					Message: "duplicate performance annotation",
					Range:   r,
				})
			}
			addons.Performance, err = p.parsePerformance()
			addons.Performance.Extend(r)
			if err != nil {
				return directives.SetRange(&addons, p.Range()), p.Annotate(err)
			}

		case "@accrue":
			if !addons.Accrual.Empty() {
				return directives.SetRange(&addons, p.Range()), p.Annotate(directives.Error{
					Message: "duplicate accrue annotation",
					Range:   r,
				})
			}
			addons.Accrual, err = p.parseAccrual()
			addons.Accrual.Extend(r)
			if err != nil {
				return directives.SetRange(&addons, p.Range()), p.Annotate(err)
			}
		}
		if _, err := p.readRestOfWhitespaceLine(); err != nil {
			return directives.SetRange(&addons, p.Range()), p.Annotate(directives.Error{})
		}
		if p.Current() != '@' {
			return directives.SetRange(&addons, p.Range()), nil
		}
	}
}

func (p *Parser) parsePerformance() (directives.Performance, error) {
	p.RangeStart("parsing performance")
	defer p.RangeEnd()
	var perf directives.Performance
	if _, err := p.ReadCharacter('('); err != nil {
		return directives.SetRange(&perf, p.Range()), p.Annotate(err)
	}
	if _, err := p.ReadWhile(isWhitespace); err != nil {
		return directives.SetRange(&perf, p.Range()), p.Annotate(err)
	}
	if p.Current() != ')' {
		if c, err := p.parseCommodity(); err != nil {
			return directives.SetRange(&perf, p.Range()), p.Annotate(err)
		} else {
			perf.Targets = append(perf.Targets, c)
		}
		if _, err := p.ReadWhile(isWhitespace); err != nil {
			return directives.SetRange(&perf, p.Range()), p.Annotate(err)
		}
	}
	for p.Current() == ',' {
		if _, err := p.ReadCharacter(','); err != nil {
			return directives.SetRange(&perf, p.Range()), p.Annotate(err)
		}
		if _, err := p.ReadWhile(isWhitespace); err != nil {
			return directives.SetRange(&perf, p.Range()), p.Annotate(err)
		}
		if c, err := p.parseCommodity(); err != nil {
			return directives.SetRange(&perf, p.Range()), p.Annotate(err)
		} else {
			perf.Targets = append(perf.Targets, c)
		}
		if _, err := p.ReadWhile(isWhitespace); err != nil {
			return directives.SetRange(&perf, p.Range()), p.Annotate(err)
		}
	}
	if _, err := p.ReadCharacter(')'); err != nil {
		return directives.SetRange(&perf, p.Range()), p.Annotate(err)
	}
	return directives.SetRange(&perf, p.Range()), nil
}

func (p *Parser) parseAccrual() (directives.Accrual, error) {
	p.RangeStart("parsing addons")
	defer p.RangeEnd()
	accrual := directives.Accrual{Range: p.Range()}
	if _, err := p.readWhitespace1(); err != nil {
		return directives.SetRange(&accrual, p.Range()), p.Annotate(err)
	}
	var err error
	if accrual.Interval, err = p.parseInterval(); err != nil {
		return directives.SetRange(&accrual, p.Range()), p.Annotate(err)
	}
	if _, err := p.readWhitespace1(); err != nil {
		return directives.SetRange(&accrual, p.Range()), p.Annotate(err)
	}
	if accrual.Start, err = p.parseDate(); err != nil {
		return directives.SetRange(&accrual, p.Range()), p.Annotate(err)
	}
	if _, err := p.readWhitespace1(); err != nil {
		return directives.SetRange(&accrual, p.Range()), p.Annotate(err)
	}
	if accrual.End, err = p.parseDate(); err != nil {
		return directives.SetRange(&accrual, p.Range()), p.Annotate(err)
	}
	if _, err := p.readWhitespace1(); err != nil {
		return directives.SetRange(&accrual, p.Range()), p.Annotate(err)
	}
	if accrual.Account, err = p.parseAccount(); err != nil {
		return directives.SetRange(&accrual, p.Range()), p.Annotate(err)
	}
	return directives.SetRange(&accrual, p.Range()), nil
}

func (p *Parser) parseInterval() (directives.Interval, error) {
	p.RangeStart("parsing interval")
	defer p.RangeEnd()
	if _, err := p.ReadAlternative([]string{"daily", "weekly", "monthly", "quarterly"}); err != nil {
		return directives.Interval{Range: p.Range()}, p.Annotate(err)
	}
	return directives.Interval{Range: p.Range()}, nil
}

func (p *Parser) readWhitespace1() (directives.Range, error) {
	p.RangeStart("")
	defer p.RangeEnd()
	if !isWhitespaceOrNewline(p.Current()) && p.Current() != scanner.EOF {
		return p.Range(), directives.Error{
			Message: fmt.Sprintf("unexpected character `%c`, want whitespace or a newline", p.Current()),
			Range:   p.Range(),
		}
	}
	return p.ReadWhile(isWhitespace)
}

func (p *Parser) readRestOfWhitespaceLine() (directives.Range, error) {
	p.RangeStart("reading the rest of the line")
	defer p.RangeEnd()
	if _, err := p.ReadWhile(isWhitespace); err != nil {
		return p.Range(), p.Annotate(err)
	}
	if p.Current() == scanner.EOF {
		return p.Range(), nil
	}
	if _, err := p.ReadCharacter('\n'); err != nil {
		return p.Range(), p.Annotate(err)
	}
	return p.Range(), nil
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

func isNewlineOrEOF(ch rune) bool {
	return ch == '\n' || ch == scanner.EOF
}
