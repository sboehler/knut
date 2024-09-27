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
	s := p.Scope("reading comment")
	if _, err := p.ReadAlternative([]string{"*", "//", "#"}); err != nil {
		return s.Range(), s.Annotate(err)
	}
	if _, err := p.ReadWhile(func(r rune) bool { return !isNewlineOrEOF(r) }); err != nil {
		return s.Range(), s.Annotate(err)
	}
	return s.Range(), nil
}

func (p *Parser) ParseFile() (directives.File, error) {
	s := p.Scope(fmt.Sprintf("parsing file `%s`", p.Path))
	var file directives.File
	for p.Current() != scanner.EOF {
		switch {

		case p.Current() == '*' || p.Current() == '#' || p.Current() == '/':
			if _, err := p.readComment(); err != nil {
				return directives.SetRange(&file, s.Range()), s.Annotate(err)
			}

		case isAlphanumeric(p.Current()) || p.Current() == '@':
			dir, err := p.parseDirective()
			file.Directives = append(file.Directives, dir)
			if err != nil {
				return directives.SetRange(&file, s.Range()), s.Annotate(err)
			}
			if p.Callback != nil {
				p.Callback(dir)
			}
		}
		if p.Current() == scanner.EOF {
			break
		}
		if _, err := p.readRestOfWhitespaceLine(); err != nil {
			return directives.SetRange(&file, s.Range()), s.Annotate(err)
		}
	}
	return directives.SetRange(&file, s.Range()), nil
}

func (p *Parser) parseDirective() (directives.Directive, error) {
	s := p.Scope("parsing directive")
	var (
		dir    directives.Directive
		addons directives.Addons
	)
	var err error
	if p.Current() == '@' {
		if addons, err = p.parseAddons(); err != nil {
			return directives.SetRange(&dir, s.Range()), s.Annotate(err)
		}
	}
	if p.Current() == 'i' {
		if dir.Directive, err = p.parseInclude(); err != nil {
			return directives.SetRange(&dir, s.Range()), s.Annotate(err)
		}
	} else {
		date, err := p.parseDate()
		if err != nil {
			return directives.SetRange(&dir, s.Range()), s.Annotate(err)
		}
		if _, err := p.readWhitespace1(); err != nil {
			return directives.SetRange(&dir, s.Range()), s.Annotate(err)
		}
		if p.Current() == '"' {
			if dir.Directive, err = p.parseTransaction(s, date, addons); err != nil {
				return directives.SetRange(&dir, s.Range()), s.Annotate(err)
			}
		} else {
			r, err := p.ReadAlternative([]string{"open", "close", "balance", "price"})
			if err != nil {
				return directives.SetRange(&dir, s.Range()), s.Annotate(err)
			}
			if _, err := p.readWhitespace1(); err != nil {
				return directives.SetRange(&dir, s.Range()), s.Annotate(err)
			}
			switch r.Extract() {
			case "open":
				if dir.Directive, err = p.parseOpen(s, date); err != nil {
					return directives.SetRange(&dir, s.Range()), s.Annotate(err)
				}
			case "close":
				if dir.Directive, err = p.parseClose(s, date); err != nil {
					return directives.SetRange(&dir, s.Range()), s.Annotate(err)
				}
			case "balance":
				if dir.Directive, err = p.parseAssertion(s, date); err != nil {
					return directives.SetRange(&dir, s.Range()), s.Annotate(err)
				}
			case "price":
				if dir.Directive, err = p.parsePrice(s, date); err != nil {
					return directives.SetRange(&dir, s.Range()), s.Annotate(err)
				}
			}
		}
	}
	return directives.SetRange(&dir, s.Range()), nil
}

func (p *Parser) parseInclude() (directives.Include, error) {
	s := p.Scope("parsing `include` statement")
	var (
		include = directives.Include{}
		err     error
	)
	if _, err := p.ReadString("include"); err != nil {
		return directives.SetRange(&include, s.Range()), s.Annotate(err)
	}
	if _, err := p.readWhitespace1(); err != nil {
		return directives.SetRange(&include, s.Range()), s.Annotate(err)
	}
	if include.IncludePath, err = p.parseQuotedString(); err != nil {
		return directives.SetRange(&include, s.Range()), s.Annotate(err)
	}
	return directives.SetRange(&include, s.Range()), nil
}

func (p *Parser) parseOpen(s scanner.Scope, date directives.Date) (directives.Open, error) {
	s.UpdateDesc("parsing `open` directive")
	var (
		open = directives.Open{Date: date}
		err  error
	)
	if open.Account, err = p.parseAccount(); err != nil {
		err = s.Annotate(err)
	}
	return directives.SetRange(&open, s.Range()), err
}

func (p *Parser) parseClose(s scanner.Scope, date directives.Date) (directives.Close, error) {
	s.UpdateDesc("parsing `close` directive")
	var (
		close = directives.Close{Date: date}
		err   error
	)
	if close.Account, err = p.parseAccount(); err != nil {
		err = s.Annotate(err)
	}
	return directives.SetRange(&close, s.Range()), err
}

func (p *Parser) parseAssertion(s scanner.Scope, date directives.Date) (directives.Assertion, error) {
	s.UpdateDesc("parsing `balance` directive")
	var (
		assertion = directives.Assertion{Date: date}
		err       error
	)
	if isNewline(p.Current()) {
		if _, err := p.readRestOfWhitespaceLine(); err != nil {
			return directives.SetRange(&assertion, s.Range()), s.Annotate(err)
		}
		for {
			bal, err := p.parseBalance()
			assertion.Balances = append(assertion.Balances, bal)
			if err != nil {
				return directives.SetRange(&assertion, s.Range()), s.Annotate(err)
			}
			if _, err := p.readRestOfWhitespaceLine(); err != nil {
				return directives.SetRange(&assertion, s.Range()), s.Annotate(err)
			}
			if isWhitespaceOrNewline(p.Current()) || p.Current() == scanner.EOF {
				break
			}
		}
	} else {
		bal, err := p.parseBalance()
		assertion.Balances = append(assertion.Balances, bal)
		if err != nil {
			return directives.SetRange(&assertion, s.Range()), s.Annotate(err)
		}
	}
	return directives.SetRange(&assertion, s.Range()), err
}

func (p *Parser) parseBalance() (directives.Balance, error) {
	s := p.Scope("parsing balance subdirective")
	var (
		balance = directives.Balance{}
		err     error
	)
	if balance.Account, err = p.parseAccount(); err != nil {
		return directives.SetRange(&balance, s.Range()), s.Annotate(err)
	}
	if _, err := p.readWhitespace1(); err != nil {
		return directives.SetRange(&balance, s.Range()), s.Annotate(err)
	}
	if balance.Quantity, err = p.parseDecimal(); err != nil {
		return directives.SetRange(&balance, s.Range()), s.Annotate(err)
	}
	if _, err := p.readWhitespace1(); err != nil {
		return directives.SetRange(&balance, s.Range()), s.Annotate(err)
	}
	if balance.Commodity, err = p.parseCommodity(); err != nil {
		err = s.Annotate(err)
	}
	return directives.SetRange(&balance, s.Range()), err
}

func (p *Parser) parsePrice(s scanner.Scope, date directives.Date) (directives.Price, error) {
	s.UpdateDesc("parsing `balance` directive")
	var (
		price = directives.Price{Date: date}
		err   error
	)
	if price.Commodity, err = p.parseCommodity(); err != nil {
		return directives.SetRange(&price, s.Range()), s.Annotate(err)
	}
	if _, err := p.readWhitespace1(); err != nil {
		return directives.SetRange(&price, s.Range()), s.Annotate(err)
	}
	if price.Price, err = p.parseDecimal(); err != nil {
		return directives.SetRange(&price, s.Range()), s.Annotate(err)
	}
	if _, err := p.readWhitespace1(); err != nil {
		return directives.SetRange(&price, s.Range()), s.Annotate(err)
	}
	if price.Target, err = p.parseCommodity(); err != nil {
		return directives.SetRange(&price, s.Range()), err
	}
	return directives.SetRange(&price, s.Range()), err
}

func (p *Parser) parseCommodity() (directives.Commodity, error) {
	var (
		commodity directives.Commodity
		err       error
	)
	s := p.Scope("parsing commodity")
	_, err = p.ReadWhile1("a letter or a digit", isAlphanumeric)
	if err != nil {
		err = s.Annotate(err)
	}
	return directives.SetRange(&commodity, s.Range()), err
}

func (p *Parser) parseDecimal() (directives.Decimal, error) {
	s := p.Scope("parsing decimal")
	if p.Current() == '-' {
		if _, err := p.ReadCharacter('-'); err != nil {
			return directives.Decimal{Range: s.Range()}, s.Annotate(err)
		}
	}
	if _, err := p.ReadWhile1("a digit", unicode.IsDigit); err != nil {
		return directives.Decimal{Range: s.Range()}, s.Annotate(err)
	}
	if p.Current() != '.' {
		return directives.Decimal{Range: s.Range()}, nil
	}
	if _, err := p.ReadCharacter('.'); err != nil {
		return directives.Decimal{Range: s.Range()}, s.Annotate(err)
	}
	if _, err := p.ReadWhile1("a digit", unicode.IsDigit); err != nil {
		return directives.Decimal{Range: s.Range()}, s.Annotate(err)
	}
	return directives.Decimal{Range: s.Range()}, nil
}

func (p *Parser) parseAccount() (directives.Account, error) {
	s := p.Scope("parsing account")
	acc := directives.Account{}
	if p.Current() == '$' {
		acc.Macro = true
		if _, err := p.ReadCharacter('$'); err != nil {
			return directives.SetRange(&acc, s.Range()), s.Annotate(err)
		}
		if _, err := p.ReadWhile1("a letter", unicode.IsLetter); err != nil {
			return directives.SetRange(&acc, s.Range()), s.Annotate(err)
		}
		return directives.SetRange(&acc, s.Range()), nil
	}
	if _, err := p.ReadWhile1("a letter or a digit", isAlphanumeric); err != nil {
		return directives.Account{Range: s.Range()}, s.Annotate(err)
	}
	for {
		if p.Current() != ':' {
			return directives.Account{Range: s.Range()}, nil
		}
		if _, err := p.ReadCharacter(':'); err != nil {
			return directives.Account{Range: s.Range()}, s.Annotate(err)
		}
		if _, err := p.ReadWhile1("a letter or a digit", isAlphanumeric); err != nil {
			return directives.Account{Range: s.Range()}, s.Annotate(err)
		}
	}
}

func (p *Parser) parseBooking() (directives.Booking, error) {
	s := p.Scope("parsing booking")
	var (
		booking directives.Booking
		err     error
	)
	if booking.Credit, err = p.parseAccount(); err != nil {
		return directives.SetRange(&booking, s.Range()), s.Annotate(err)
	}
	if _, err := p.ReadWhile1("whitespace", isWhitespace); err != nil {
		return directives.SetRange(&booking, s.Range()), s.Annotate(err)
	}
	if booking.Debit, err = p.parseAccount(); err != nil {
		return directives.SetRange(&booking, s.Range()), s.Annotate(err)
	}
	if _, err := p.ReadWhile1("whitespace", isWhitespace); err != nil {
		return directives.SetRange(&booking, s.Range()), s.Annotate(err)
	}
	if booking.Quantity, err = p.parseDecimal(); err != nil {
		return directives.SetRange(&booking, s.Range()), s.Annotate(err)
	}
	if _, err := p.ReadWhile1("whitespace", isWhitespace); err != nil {
		return directives.SetRange(&booking, s.Range()), s.Annotate(err)
	}
	if booking.Commodity, err = p.parseCommodity(); err != nil {
		return directives.SetRange(&booking, s.Range()), s.Annotate(err)
	}
	return directives.SetRange(&booking, s.Range()), nil
}

func (p *Parser) parseDate() (directives.Date, error) {
	s := p.Scope("parsing the date")

	for i := 0; i < 4; i++ {
		if _, err := p.ReadCharacterWith("a digit", unicode.IsDigit); err != nil {
			return directives.Date{Range: s.Range()}, s.Annotate(err)
		}
	}
	for i := 0; i < 2; i++ {
		if _, err := p.ReadCharacter('-'); err != nil {
			return directives.Date{Range: s.Range()}, s.Annotate(err)
		}
		for j := 0; j < 2; j++ {
			if _, err := p.ReadCharacterWith("a digit", unicode.IsDigit); err != nil {
				return directives.Date{Range: s.Range()}, s.Annotate(err)
			}
		}
	}
	return directives.Date{Range: s.Range()}, nil
}

func (p *Parser) parseQuotedString() (directives.QuotedString, error) {
	s := p.Scope("parsing quoted string")
	var (
		qs  directives.QuotedString
		err error
	)
	if _, err := p.ReadCharacter('"'); err != nil {
		return directives.SetRange(&qs, s.Range()), s.Annotate(err)
	}
	if qs.Content, err = p.ReadWhile(func(r rune) bool { return r != '"' }); err != nil {
		return directives.SetRange(&qs, s.Range()), s.Annotate(err)
	}
	if _, err := p.ReadCharacter('"'); err != nil {
		return directives.SetRange(&qs, s.Range()), s.Annotate(err)
	}
	return directives.SetRange(&qs, s.Range()), nil
}

func (p *Parser) parseTransaction(s scanner.Scope, date directives.Date, addons directives.Addons) (directives.Transaction, error) {
	s.UpdateDesc("parsing transaction")
	var (
		trx = directives.Transaction{Date: date, Addons: addons}
		err error
	)
	if trx.Description, err = p.parseQuotedString(); err != nil {
		return directives.SetRange(&trx, s.Range()), s.Annotate(err)
	}
	if _, err := p.readRestOfWhitespaceLine(); err != nil {
		return directives.SetRange(&trx, s.Range()), s.Annotate(err)
	}
	for {
		b, err := p.parseBooking()
		trx.Bookings = append(trx.Bookings, b)
		if err != nil {
			return directives.SetRange(&trx, s.Range()), s.Annotate(err)
		}
		if _, err := p.readRestOfWhitespaceLine(); err != nil {
			return directives.SetRange(&trx, s.Range()), s.Annotate(err)
		}
		if isWhitespaceOrNewline(p.Current()) || p.Current() == scanner.EOF {
			break
		}
	}
	return directives.SetRange(&trx, s.Range()), nil
}

func (p *Parser) parseAddons() (directives.Addons, error) {
	s := p.Scope("parsing addons")
	var addons directives.Addons
	for {
		r, err := p.ReadAlternative([]string{"@performance", "@accrue"})
		if err != nil {
			return directives.SetRange(&addons, r), s.Annotate(err)
		}
		switch r.Extract() {
		case "@performance":
			if !addons.Performance.Empty() {
				return directives.SetRange(&addons, s.Range()), s.Annotate(directives.Error{
					Message: "duplicate performance annotation",
					Range:   r,
				})
			}
			addons.Performance, err = p.parsePerformance()
			addons.Performance.Extend(r)
			if err != nil {
				return directives.SetRange(&addons, s.Range()), s.Annotate(err)
			}

		case "@accrue":
			if !addons.Accrual.Empty() {
				return directives.SetRange(&addons, s.Range()), s.Annotate(directives.Error{
					Message: "duplicate accrue annotation",
					Range:   r,
				})
			}
			addons.Accrual, err = p.parseAccrual()
			addons.Accrual.Extend(r)
			if err != nil {
				return directives.SetRange(&addons, s.Range()), s.Annotate(err)
			}
		}
		if _, err := p.readRestOfWhitespaceLine(); err != nil {
			return directives.SetRange(&addons, s.Range()), s.Annotate(directives.Error{})
		}
		if p.Current() != '@' {
			return directives.SetRange(&addons, s.Range()), nil
		}
	}
}

func (p *Parser) parsePerformance() (directives.Performance, error) {
	s := p.Scope("parsing performance")
	var perf directives.Performance
	if _, err := p.ReadCharacter('('); err != nil {
		return directives.SetRange(&perf, s.Range()), s.Annotate(err)
	}
	if _, err := p.ReadWhile(isWhitespace); err != nil {
		return directives.SetRange(&perf, s.Range()), s.Annotate(err)
	}
	if p.Current() != ')' {
		if c, err := p.parseCommodity(); err != nil {
			return directives.SetRange(&perf, s.Range()), s.Annotate(err)
		} else {
			perf.Targets = append(perf.Targets, c)
		}
		if _, err := p.ReadWhile(isWhitespace); err != nil {
			return directives.SetRange(&perf, s.Range()), s.Annotate(err)
		}
	}
	for p.Current() == ',' {
		if _, err := p.ReadCharacter(','); err != nil {
			return directives.SetRange(&perf, s.Range()), s.Annotate(err)
		}
		if _, err := p.ReadWhile(isWhitespace); err != nil {
			return directives.SetRange(&perf, s.Range()), s.Annotate(err)
		}
		if c, err := p.parseCommodity(); err != nil {
			return directives.SetRange(&perf, s.Range()), s.Annotate(err)
		} else {
			perf.Targets = append(perf.Targets, c)
		}
		if _, err := p.ReadWhile(isWhitespace); err != nil {
			return directives.SetRange(&perf, s.Range()), s.Annotate(err)
		}
	}
	if _, err := p.ReadCharacter(')'); err != nil {
		return directives.SetRange(&perf, s.Range()), s.Annotate(err)
	}
	return directives.SetRange(&perf, s.Range()), nil
}

func (p *Parser) parseAccrual() (directives.Accrual, error) {
	s := p.Scope("parsing addons")
	accrual := directives.Accrual{Range: s.Range()}
	if _, err := p.readWhitespace1(); err != nil {
		return directives.SetRange(&accrual, s.Range()), s.Annotate(err)
	}
	var err error
	if accrual.Interval, err = p.parseInterval(); err != nil {
		return directives.SetRange(&accrual, s.Range()), s.Annotate(err)
	}
	if _, err := p.readWhitespace1(); err != nil {
		return directives.SetRange(&accrual, s.Range()), s.Annotate(err)
	}
	if accrual.Start, err = p.parseDate(); err != nil {
		return directives.SetRange(&accrual, s.Range()), s.Annotate(err)
	}
	if _, err := p.readWhitespace1(); err != nil {
		return directives.SetRange(&accrual, s.Range()), s.Annotate(err)
	}
	if accrual.End, err = p.parseDate(); err != nil {
		return directives.SetRange(&accrual, s.Range()), s.Annotate(err)
	}
	if _, err := p.readWhitespace1(); err != nil {
		return directives.SetRange(&accrual, s.Range()), s.Annotate(err)
	}
	if accrual.Account, err = p.parseAccount(); err != nil {
		return directives.SetRange(&accrual, s.Range()), s.Annotate(err)
	}
	return directives.SetRange(&accrual, s.Range()), nil
}

func (p *Parser) parseInterval() (directives.Interval, error) {
	s := p.Scope("parsing interval")
	if _, err := p.ReadAlternative([]string{"daily", "weekly", "monthly", "quarterly"}); err != nil {
		return directives.Interval{Range: s.Range()}, s.Annotate(err)
	}
	return directives.Interval{Range: s.Range()}, nil
}

func (p *Parser) readWhitespace1() (directives.Range, error) {
	s := p.Scope("")
	if !isWhitespaceOrNewline(p.Current()) && p.Current() != scanner.EOF {
		return s.Range(), directives.Error{
			Message: fmt.Sprintf("unexpected character `%c`, want whitespace or a newline", p.Current()),
			Range:   s.Range(),
		}
	}
	return p.ReadWhile(isWhitespace)
}

func (p *Parser) readRestOfWhitespaceLine() (directives.Range, error) {
	s := p.Scope("reading the rest of the line")
	if _, err := p.ReadWhile(isWhitespace); err != nil {
		return s.Range(), s.Annotate(err)
	}
	if p.Current() == scanner.EOF {
		return s.Range(), nil
	}
	if _, err := p.ReadCharacter('\n'); err != nil {
		return s.Range(), s.Annotate(err)
	}
	return s.Range(), nil
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
