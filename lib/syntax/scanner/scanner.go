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
	"unicode/utf8"

	"github.com/sboehler/knut/lib/syntax"
)

type Range = syntax.Range

// Scanner is a scanner.
type Scanner struct {
	text string
	path string

	// current contains the current rune
	current    rune
	currentLen int
	offset     int

	ranges []Range
}

// New creates a new Scanner.
func New(text, path string) *Scanner {
	return &Scanner{
		text: text,
		path: path,
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
	if l := len(s.ranges); l > 0 {
		s.ranges[l-1].End = s.offset
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
			return syntax.Error{
				Message: "unexpected end of file",
				Range: Range{
					Start: s.Offset(),
					End:   s.Offset(),
					Path:  s.path,
					Text:  s.text,
				},
			}
		case 1:
			return syntax.Error{
				Message: "invalid unicode character",
				Range: Range{
					Start: s.Offset(),
					End:   s.Offset(),
					Path:  s.path,
					Text:  s.text,
				},
			}
		}
	}
	return nil
}

func (s *Scanner) Start() {
	s.ranges = append(s.ranges, Range{
		Start: s.Offset(),
		End:   s.Offset(),
		Path:  s.path,
		Text:  s.text,
	})
}

func (s *Scanner) End() Range {
	last := len(s.ranges) - 1
	r := s.ranges[last]
	s.ranges = s.ranges[:last]
	if last > 0 {
		s.ranges[last-1].End = r.End
	}
	return r
}

// EOF is a rune representing the end of a file
const EOF = rune(-1)

// ReadWhile reads a string while the predicate holds.
func (s *Scanner) ReadWhile(pred func(r rune) bool) (Range, error) {
	s.Start()
	defer s.End()
	for pred(s.Current()) && s.Current() != EOF {
		if err := s.Advance(); err != nil {
			return s.Range(), syntax.Error{
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
func (s *Scanner) ReadWhile1(pred func(r rune) bool) (Range, error) {
	s.Start()
	defer s.End()
	if s.Current() == EOF {
		return s.Range(), syntax.Error{
			Message: "unexpected end of file",
			Range:   s.Range(),
		}
	}
	if !pred(s.Current()) {
		return s.Range(), syntax.Error{
			Message: fmt.Sprintf("unexpected character %c", s.Current()),
			Range:   s.Range(),
		}
	}
	for pred(s.Current()) && s.Current() != EOF {
		if err := s.Advance(); err != nil {
			return s.Range(), syntax.Error{
				Message: "reading next character",
				Range:   s.Range(),
				Wrapped: err,
			}
		}
	}
	return s.Range(), nil
}

// ReadUntil advances the scanner until the predicate holds.
func (s *Scanner) ReadUntil(pred func(r rune) bool) (Range, error) {
	s.Start()
	defer s.End()
	for !pred(s.Current()) {
		if err := s.Advance(); err != nil {
			return s.Range(), syntax.Error{
				Message: "reading next character",
				Range:   s.Range(),
				Wrapped: err,
			}
		}
		if s.Current() == EOF {
			return s.Range(), syntax.Error{
				Message: "unexpected end of file",
				Range:   s.Range(),
			}
		}
	}
	return s.Range(), nil
}

// ReadCharacter consumes the given rune.
func (s *Scanner) ReadCharacter(r rune) (Range, error) {
	s.Start()
	defer s.End()
	if s.Current() == EOF {
		return s.Range(), syntax.Error{
			Message: fmt.Sprintf("unexpected end of file, want %c", r),
			Range:   s.Range(),
		}
	}
	if s.Current() != r {
		return s.Range(), syntax.Error{
			Message: fmt.Sprintf("unexpected character %c, want %c", s.current, r),
			Range:   s.Range(),
		}
	}
	if err := s.Advance(); err != nil {
		return s.Range(), syntax.Error{
			Message: "reading next character",
			Range:   s.Range(),
			Wrapped: err,
		}
	}
	return s.Range(), nil
}

// ReadCharacter consume a rune satisfying the predicate.
func (s *Scanner) ReadCharacterWith(pred func(rune) bool) (Range, error) {
	s.Start()
	defer s.End()
	if s.Current() == EOF {
		return s.Range(), syntax.Error{
			Message: "unexpected end of file",
			Range:   s.Range(),
		}
	}
	if !pred(s.Current()) {
		return s.Range(), syntax.Error{
			Message: fmt.Sprintf("unexpected character: %c", s.Current()),
			Range:   s.Range(),
		}
	}
	if err := s.Advance(); err != nil {
		return s.Range(), syntax.Error{
			Message: "reading next character",
			Range:   s.Range(),
			Wrapped: err,
		}
	}
	return s.Range(), nil
}

// ReadString parses the given string.
func (s *Scanner) ReadString(str string) (Range, error) {
	s.Start()
	defer s.End()
	for _, ch := range str {
		if ch != s.Current() {
			return s.Range(), syntax.Error{
				Message: fmt.Sprintf("while reading %q", str),
				Range:   s.Range(),
			}
		}
		if err := s.Advance(); err != nil {
			return s.Range(), syntax.Error{
				Message: fmt.Sprintf("while reading %q", str),
				Range:   s.Range(),
				Wrapped: err,
			}
		}
	}
	return s.Range(), nil
}

// ReadN reads a string with n runes.
func (s *Scanner) ReadN(n int) (Range, error) {
	s.Start()
	defer s.End()
	for i := 0; i < n; i++ {
		if s.current == EOF {
			return s.Range(), syntax.Error{
				Range:   s.Range(),
				Message: fmt.Sprintf("while reading %d of %d characters", i, n),
				Wrapped: io.EOF,
			}
		}
		if err := s.Advance(); err != nil {
			return s.Range(), syntax.Error{
				Range:   s.Range(),
				Message: fmt.Sprintf("while reading %d of %d characters", i, n),
				Wrapped: err,
			}
		}
	}
	return s.Range(), nil
}

func (s *Scanner) Range() Range {
	return s.ranges[len(s.ranges)-1]
}
