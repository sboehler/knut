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

package scanner

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/sboehler/knut/lib/model/accounts"
	"github.com/sboehler/knut/lib/model/commodities"

	"github.com/shopspring/decimal"
)

// ReadQuotedString parses a quoted string
func ReadQuotedString(b *Scanner) (string, error) {
	if err := b.ConsumeRune('"'); err != nil {
		return "", err
	}
	s, err := b.ReadWhile(func(r rune) bool {
		return r != '"'
	})
	if err != nil {
		return s, err
	}
	if err := b.ConsumeRune('"'); err != nil {
		return s, err
	}
	return s, nil
}

// ParseIdentifier parses an identifier
func ParseIdentifier(b *Scanner) (string, error) {
	s := strings.Builder{}
	for unicode.IsLetter(b.Current()) || unicode.IsDigit(b.Current()) {
		s.WriteRune(b.Current())
		if err := b.Advance(); err != nil {
			return s.String(), err
		}
	}
	return s.String(), nil
}

// ParseAccount parses an account
func ParseAccount(b *Scanner) (*accounts.Account, error) {
	name := strings.Builder{}
	for {
		i, err := ParseIdentifier(b)
		if err != nil {
			return nil, err
		}
		if _, err := name.WriteString(i); err != nil {
			return nil, err
		}
		if b.Current() != ':' {
			return accounts.Get(name.String())
		}
		if err := b.ConsumeRune(':'); err != nil {
			return nil, err
		}
		if _, err := name.WriteRune(':'); err != nil {
			return nil, err
		}
	}
}

// ParseAccountType parses an account type
func ParseAccountType(b *Scanner) (accounts.AccountType, error) {
	s, err := b.ReadWhile(unicode.IsLetter)
	if err != nil {
		return 0, err
	}
	switch s {
	case "TBD":
		return accounts.TBD, nil
	case "Assets":
		return accounts.ASSETS, nil
	case "Liabilities":
		return accounts.LIABILITIES, nil
	case "Equity":
		return accounts.EQUITY, nil
	case "Income":
		return accounts.INCOME, nil
	case "Expenses":
		return accounts.EXPENSES, nil
	default:
		return 0, fmt.Errorf("Expected account type, got %v", s)
	}
}

// ParseDecimal parses a decimal number
func ParseDecimal(p *Scanner) (decimal.Decimal, error) {
	b := strings.Builder{}
	for unicode.IsDigit(p.Current()) || p.Current() == '.' || p.Current() == '-' {
		b.WriteRune(p.Current())
		if err := p.Advance(); err != nil {
			return decimal.Zero, err
		}
	}
	d, err := decimal.NewFromString(b.String())
	if err != nil {
		return decimal.Zero, err
	}
	return d, nil
}

// ParseDate parses a date as YYYY-MM-DD
func ParseDate(p *Scanner) (time.Time, error) {
	s, err := ReadString(p, 10)
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse("2006-01-02", s)
}

// ReadString reads a string with n runes
func ReadString(p *Scanner, n int) (string, error) {
	b := strings.Builder{}
	for i := 0; i < n; i++ {
		b.WriteRune(p.Current())
		if err := p.Advance(); err != nil {
			return "", err
		}
	}
	return b.String(), nil
}

// ParseFloat parses a floating point number
func ParseFloat(p *Scanner) (float64, error) {
	b := strings.Builder{}
	for unicode.IsDigit(p.Current()) || p.Current() == '.' || p.Current() == '-' {
		b.WriteRune(p.Current())
		if err := p.Advance(); err != nil {
			return 0, err
		}
	}
	return strconv.ParseFloat(b.String(), 64)
}

// ParseCommodity parses a commodity
func ParseCommodity(p *Scanner) (*commodities.Commodity, error) {
	i, err := ParseIdentifier(p)
	if err != nil {
		return commodities.Get(""), err
	}
	return commodities.Get(i), nil
}
