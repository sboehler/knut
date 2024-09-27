// Copyright 2021 Silvio Böhler
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
	"io"
	"strings"
	"unicode/utf8"

	"github.com/sboehler/knut/lib/syntax/directives"
)

type Range = directives.Range

// Scanner is a scanner.
type Scanner struct {
	text string
	Path string

	// current contains the current rune
	current    rune
	currentLen int
	offset     int
}

type Scope struct {
	Desc    string
	Start   int
	Scanner *Scanner
}

func (s *Scope) UpdateDesc(desc string) {
	s.Desc = desc
}

func (s *Scope) Range() directives.Range {
	return directives.Range{
		Text:  s.Scanner.text,
		Path:  s.Scanner.Path,
		Start: s.Start,
		End:   s.Scanner.offset,
	}
}

func (s Scope) Annotate(err error) error {
	return directives.Error{
		Message: "while " + s.Desc,
		Range:   s.Range(),
		Wrapped: err,
	}
}

// New creates a new Scanner.
func New(text, path string) *Scanner {
	return &Scanner{
		text: text,
		Path: path,
	}
}

// Current returns the current rune.
func (s *Scanner) Current() rune {
	return s.current
}

// Offset returns the current offset.
func (s *Scanner) Offset() int {
	return s.offset
}

// Advance reads a rune.
func (s *Scanner) Advance() error {
	s.offset += s.currentLen
	if s.offset == len(s.text) && s.current != EOF {
		s.current = EOF
		s.currentLen = 0
		return nil
	}
	s.current, s.currentLen = utf8.DecodeRuneInString(s.text[s.offset:])
	if s.current == utf8.RuneError {
		switch s.currentLen {
		case 0:
			return directives.Error{
				Message: "unexpected end of file",
				Range: Range{
					Start: s.Offset(),
					End:   s.Offset(),
					Path:  s.Path,
					Text:  s.text,
				},
			}
		case 1:
			return directives.Error{
				Message: "invalid unicode character",
				Range: Range{
					Start: s.Offset(),
					End:   s.Offset(),
					Path:  s.Path,
					Text:  s.text,
				},
			}
		}
	}
	return nil
}

func (s *Scanner) Scope(desc string) Scope {
	scope := Scope{
		Desc:    desc,
		Start:   s.offset,
		Scanner: s,
	}
	return scope
}

func (s *Scanner) Backtrack(offset int) {
	s.offset = offset
	s.current, s.currentLen = utf8.DecodeRuneInString(s.text[s.offset:])
}

// EOF is a rune representing the end of a file
const EOF = rune(-1)

// ReadWhile reads a string while the predicate holds.
func (s *Scanner) ReadWhile(pred func(r rune) bool) (Range, error) {
	sc := s.Scope("")
	for pred(s.Current()) && s.Current() != EOF {
		if err := s.Advance(); err != nil {
			return sc.Range(), directives.Error{
				Message: "reading next character",
				Range:   sc.Range(),
				Wrapped: err,
			}
		}
	}
	return sc.Range(), nil
}

// ReadWhile reads a string while the predicate holds. The predicate must be
// satisfied at least once.
func (s *Scanner) ReadWhile1(desc string, pred func(r rune) bool) (Range, error) {
	sc := s.Scope("")
	if s.Current() == EOF {
		return sc.Range(), directives.Error{
			Message: fmt.Sprintf("unexpected end of file, want %s", desc),
			Range:   sc.Range(),
		}
	}
	if !pred(s.Current()) {
		return sc.Range(), directives.Error{
			Message: fmt.Sprintf("unexpected character `%c`, want %s", s.Current(), desc),
			Range:   sc.Range(),
		}
	}
	for pred(s.Current()) && s.Current() != EOF {
		if err := s.Advance(); err != nil {
			return sc.Range(), directives.Error{
				Message: "reading next character",
				Range:   sc.Range(),
				Wrapped: err,
			}
		}
	}
	return sc.Range(), nil
}

// ReadUntil advances the scanner until the predicate holds.
func (s *Scanner) ReadUntil(desc string, pred func(r rune) bool) (Range, error) {
	sc := s.Scope("")
	for !pred(s.Current()) {
		if err := s.Advance(); err != nil {
			return sc.Range(), directives.Error{
				Message: "reading next character",
				Range:   sc.Range(),
				Wrapped: err,
			}
		}
		if s.Current() == EOF {
			return sc.Range(), directives.Error{
				Message: fmt.Sprintf("unexpected end of file, want %s", desc),
				Range:   sc.Range(),
			}
		}
	}
	return sc.Range(), nil
}

// ReadCharacter consumes the given rune.
func (s *Scanner) ReadCharacter(r rune) (Range, error) {
	sc := s.Scope("")
	if s.Current() == EOF {
		return sc.Range(), directives.Error{
			Message: fmt.Sprintf("unexpected end of file, want `%c`", r),
			Range:   sc.Range(),
		}
	}
	if s.Current() != r {
		return sc.Range(), directives.Error{
			Message: fmt.Sprintf("unexpected character `%c`, want `%c`", s.current, r),
			Range:   sc.Range(),
		}
	}
	if err := s.Advance(); err != nil {
		return sc.Range(), directives.Error{
			Message: "reading next character",
			Range:   sc.Range(),
			Wrapped: err,
		}
	}
	return sc.Range(), nil
}

// ReadCharacter consume a rune satisfying the predicate.
func (s *Scanner) ReadCharacterWith(desc string, pred func(rune) bool) (Range, error) {
	sc := s.Scope("")
	if s.Current() == EOF {
		return sc.Range(), directives.Error{
			Message: fmt.Sprintf("unexpected end of file, want %s", desc),
			Range:   sc.Range(),
		}
	}
	if !pred(s.Current()) {
		return sc.Range(), directives.Error{
			Message: fmt.Sprintf("unexpected character `%c`, want %s", s.Current(), desc),
			Range:   sc.Range(),
		}
	}
	if err := s.Advance(); err != nil {
		return sc.Range(), directives.Error{
			Message: "reading next character",
			Range:   sc.Range(),
			Wrapped: err,
		}
	}
	return sc.Range(), nil
}

// ReadString parses the given string.
func (s *Scanner) ReadString(str string) (Range, error) {
	sc := s.Scope("")
	for _, ch := range str {
		if ch != s.Current() {
			return sc.Range(), directives.Error{
				Message: fmt.Sprintf("while reading %q", str),
				Range:   sc.Range(),
			}
		}
		if err := s.Advance(); err != nil {
			return sc.Range(), directives.Error{
				Message: fmt.Sprintf("while reading %q", str),
				Range:   sc.Range(),
				Wrapped: err,
			}
		}
	}
	return sc.Range(), nil
}

func (s *Scanner) ReadAlternative(ss []string) (Range, error) {
	sc := s.Scope("")
	if s.current == EOF {
		return sc.Range(), directives.Error{
			Message: fmt.Sprintf("unexpected end of file, want one of %s", format(ss)),
			Range:   sc.Range(),
		}
	}
	var end int
	for _, t := range ss {
		r, err := s.ReadString(t)
		if err == nil {
			return r, nil
		}
		if r.End > end {
			end = r.End
		}
		s.Backtrack(sc.Start)
	}
	return sc.Range(), directives.Error{
		Message: fmt.Sprintf("unexpected input, want one of %s", format(ss)),
		Range:   sc.Range(),
	}
}

func format(ss []string) string {
	var b strings.Builder
	b.WriteString("{")
	for i, s := range ss {
		if i != 0 {
			b.WriteString(", ")
		}
		b.WriteString("`")
		b.WriteString(s)
		b.WriteString("`")
	}
	b.WriteString("}")
	return b.String()

}

// ReadN reads a string with n runes.
func (s *Scanner) ReadN(n int) (Range, error) {
	sc := s.Scope("")
	for i := 0; i < n; i++ {
		if s.current == EOF {
			return sc.Range(), directives.Error{
				Range:   sc.Range(),
				Message: fmt.Sprintf("while reading %d of %d characters", i, n),
				Wrapped: io.EOF,
			}
		}
		if err := s.Advance(); err != nil {
			return sc.Range(), directives.Error{
				Range:   sc.Range(),
				Message: fmt.Sprintf("while reading %d of %d characters", i, n),
				Wrapped: err,
			}
		}
	}
	return sc.Range(), nil
}
