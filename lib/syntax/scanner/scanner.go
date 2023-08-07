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

	scopes []scope
}

type scope struct {
	Range directives.Range
	Desc  string
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
	if l := len(s.scopes); l > 0 {
		s.scopes[l-1].Range.End = s.offset
	}
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

func (s *Scanner) RangeStart(desc string) {
	s.scopes = append(s.scopes, scope{
		Range: Range{Start: s.Offset(), End: s.Offset(), Path: s.Path, Text: s.text},
		Desc:  desc,
	})
}

func (s *Scanner) RangeContinue(desc string) {
	if len(s.scopes) == 0 {
		s.RangeStart(desc)
		return
	}
	s.scopes = append(s.scopes, scope{
		Range: s.scopes[len(s.scopes)-1].Range,
		Desc:  desc,
	})
}

func (s *Scanner) Backtrack() {
	s.offset = s.scopes[len(s.scopes)-1].Range.Start
	s.current, s.currentLen = utf8.DecodeRuneInString(s.text[s.offset:])

}

func (s *Scanner) RangeEnd() {
	last := len(s.scopes) - 1
	if last > 0 {
		s.scopes[last-1].Range.End = s.scopes[last].Range.End
	}
	s.scopes = s.scopes[:last]
}

func (s *Scanner) Annotate(err error) error {
	return directives.Error{
		Message: "while " + s.scopes[len(s.scopes)-1].Desc,
		Range:   s.Range(),
		Wrapped: err,
	}
}

// EOF is a rune representing the end of a file
const EOF = rune(-1)

// ReadWhile reads a string while the predicate holds.
func (s *Scanner) ReadWhile(pred func(r rune) bool) (Range, error) {
	s.RangeStart("")
	defer s.RangeEnd()
	for pred(s.Current()) && s.Current() != EOF {
		if err := s.Advance(); err != nil {
			return s.Range(), directives.Error{
				Message: "reading next character",
				Range:   s.Range(),
				Wrapped: err,
			}
		}
	}
	return s.Range(), nil
}

// ReadWhile reads a string while the predicate holds. The predicate must be
// satisfied at least once.
func (s *Scanner) ReadWhile1(desc string, pred func(r rune) bool) (Range, error) {
	s.RangeStart("")
	defer s.RangeEnd()
	if s.Current() == EOF {
		return s.Range(), directives.Error{
			Message: fmt.Sprintf("unexpected end of file, want %s", desc),
			Range:   s.Range(),
		}
	}
	if !pred(s.Current()) {
		return s.Range(), directives.Error{
			Message: fmt.Sprintf("unexpected character `%c`, want %s", s.Current(), desc),
			Range:   s.Range(),
		}
	}
	for pred(s.Current()) && s.Current() != EOF {
		if err := s.Advance(); err != nil {
			return s.Range(), directives.Error{
				Message: "reading next character",
				Range:   s.Range(),
				Wrapped: err,
			}
		}
	}
	return s.Range(), nil
}

// ReadUntil advances the scanner until the predicate holds.
func (s *Scanner) ReadUntil(desc string, pred func(r rune) bool) (Range, error) {
	s.RangeStart("")
	defer s.RangeEnd()
	for !pred(s.Current()) {
		if err := s.Advance(); err != nil {
			return s.Range(), directives.Error{
				Message: "reading next character",
				Range:   s.Range(),
				Wrapped: err,
			}
		}
		if s.Current() == EOF {
			return s.Range(), directives.Error{
				Message: fmt.Sprintf("unexpected end of file, want %s", desc),
				Range:   s.Range(),
			}
		}
	}
	return s.Range(), nil
}

// ReadCharacter consumes the given rune.
func (s *Scanner) ReadCharacter(r rune) (Range, error) {
	s.RangeStart("")
	defer s.RangeEnd()
	if s.Current() == EOF {
		return s.Range(), directives.Error{
			Message: fmt.Sprintf("unexpected end of file, want `%c`", r),
			Range:   s.Range(),
		}
	}
	if s.Current() != r {
		return s.Range(), directives.Error{
			Message: fmt.Sprintf("unexpected character `%c`, want `%c`", s.current, r),
			Range:   s.Range(),
		}
	}
	if err := s.Advance(); err != nil {
		return s.Range(), directives.Error{
			Message: "reading next character",
			Range:   s.Range(),
			Wrapped: err,
		}
	}
	return s.Range(), nil
}

// ReadCharacter consume a rune satisfying the predicate.
func (s *Scanner) ReadCharacterWith(desc string, pred func(rune) bool) (Range, error) {
	s.RangeStart("")
	defer s.RangeEnd()
	if s.Current() == EOF {
		return s.Range(), directives.Error{
			Message: fmt.Sprintf("unexpected end of file, want %s", desc),
			Range:   s.Range(),
		}
	}
	if !pred(s.Current()) {
		return s.Range(), directives.Error{
			Message: fmt.Sprintf("unexpected character `%c`, want %s", s.Current(), desc),
			Range:   s.Range(),
		}
	}
	if err := s.Advance(); err != nil {
		return s.Range(), directives.Error{
			Message: "reading next character",
			Range:   s.Range(),
			Wrapped: err,
		}
	}
	return s.Range(), nil
}

// ReadString parses the given string.
func (s *Scanner) ReadString(str string) (Range, error) {
	s.RangeStart("")
	defer s.RangeEnd()
	for _, ch := range str {
		if ch != s.Current() {
			return s.Range(), directives.Error{
				Message: fmt.Sprintf("while reading %q", str),
				Range:   s.Range(),
			}
		}
		if err := s.Advance(); err != nil {
			return s.Range(), directives.Error{
				Message: fmt.Sprintf("while reading %q", str),
				Range:   s.Range(),
				Wrapped: err,
			}
		}
	}
	return s.Range(), nil
}

func (s *Scanner) ReadAlternative(ss []string) (Range, error) {
	s.RangeStart("")
	defer s.RangeEnd()
	if s.current == EOF {
		return s.Range(), directives.Error{
			Message: fmt.Sprintf("unexpected end of file, want one of %s", format(ss)),
			Range:   s.Range(),
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
		s.Backtrack()
	}
	return s.Range(), directives.Error{
		Message: fmt.Sprintf("unexpected input, want one of %s", format(ss)),
		Range:   s.Range(),
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
	s.RangeStart("")
	defer s.RangeEnd()
	for i := 0; i < n; i++ {
		if s.current == EOF {
			return s.Range(), directives.Error{
				Range:   s.Range(),
				Message: fmt.Sprintf("while reading %d of %d characters", i, n),
				Wrapped: io.EOF,
			}
		}
		if err := s.Advance(); err != nil {
			return s.Range(), directives.Error{
				Range:   s.Range(),
				Message: fmt.Sprintf("while reading %d of %d characters", i, n),
				Wrapped: err,
			}
		}
	}
	return s.Range(), nil
}

func (s *Scanner) Range() Range {
	return s.scopes[len(s.scopes)-1].Range
}
