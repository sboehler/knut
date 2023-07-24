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
				Range:   s.Range(),
			}
		case 1:
			return syntax.Error{
				Message: "invalid unicode character",
				Range:   s.Range(),
			}
		}
	}
	return nil
}

// EOF is a rune representing the end of a file
const EOF = rune(-1)

// ReadWhile reads a string while the predicate holds.
func (s *Scanner) ReadWhile(pred func(r rune) bool) (Range, error) {
	rng := s.Range()
	for pred(s.Current()) && s.Current() != EOF {
		if err := s.Advance(); err != nil {
			rng.SetEnd(s.Offset())
			return rng, syntax.Error{
				Message: "reading next character",
				Range:   rng,
				Wrapped: err,
			}
		}
	}
	rng.SetEnd(s.Offset())
	return rng, nil
}

// ReadWhile reads a string while the predicate holds. The predicate must be
// satisfied at least once.
func (s *Scanner) ReadWhile1(pred func(r rune) bool) (Range, error) {
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
	rng := s.Range()
	for pred(s.Current()) && s.Current() != EOF {
		if err := s.Advance(); err != nil {
			rng.SetEnd(s.Offset())
			return rng, syntax.Error{
				Message: "reading next character",
				Range:   rng,
				Wrapped: err,
			}
		}
	}
	return Done(s.Offset(), &rng), nil
}

// ReadUntil advances the scanner until the predicate holds.
func (s *Scanner) ReadUntil(pred func(r rune) bool) (Range, error) {
	rng := s.Range()
	for !pred(s.Current()) {
		if err := s.Advance(); err != nil {
			rng.SetEnd(s.Offset())
			return rng, syntax.Error{
				Message: "reading next character",
				Range:   rng,
				Wrapped: err,
			}
		}
		if s.Current() == EOF {
			rng.SetEnd(s.Offset())
			return rng, syntax.Error{
				Message: "unexpected end of file",
				Range:   rng,
			}
		}
	}
	return Done(s.Offset(), &rng), nil
}

// ReadCharacter consumes the given rune.
func (s *Scanner) ReadCharacter(r rune) (Range, error) {
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
	rng := s.Range()
	if err := s.Advance(); err != nil {
		rng.SetEnd(s.Offset())
		return rng, syntax.Error{
			Message: "reading next character",
			Range:   rng,
			Wrapped: err,
		}
	}
	rng.SetEnd(s.Offset())
	return rng, nil
}

// ReadCharacter consume a rune satisfying the predicate.
func (s *Scanner) ReadCharacterWith(pred func(rune) bool) (Range, error) {
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
	rng := s.Range()
	if err := s.Advance(); err != nil {
		return rng, syntax.Error{
			Message: "reading next character",
			Range:   rng,
			Wrapped: err,
		}
	}
	rng.SetEnd(s.Offset())
	return rng, nil
}

// ReadString parses the given string.
func (s *Scanner) ReadString(str string) (Range, error) {
	rng := s.Range()
	for _, ch := range str {
		if ch != s.Current() {
			rng.SetEnd(s.Offset())
			return rng, syntax.Error{
				Message: fmt.Sprintf("while reading %q", str),
				Range:   rng,
			}
		}
		if err := s.Advance(); err != nil {
			rng.SetEnd(s.Offset())
			return rng, syntax.Error{
				Message: fmt.Sprintf("while reading %q", str),
				Range:   rng,
				Wrapped: err,
			}
		}
	}
	rng.SetEnd(s.Offset())
	return rng, nil
}

// ReadN reads a string with n runes.
func (s *Scanner) ReadN(n int) (Range, error) {
	rng := s.Range()
	for i := 0; i < n; i++ {
		if s.current == EOF {
			rng.SetEnd(s.Offset())
			return rng, syntax.Error{
				Range:   rng,
				Message: fmt.Sprintf("while reading %d of %d characters", i, n),
				Wrapped: io.EOF,
			}
		}
		if err := s.Advance(); err != nil {
			rng.SetEnd(s.Offset())
			return rng, syntax.Error{
				Range:   rng,
				Message: fmt.Sprintf("while reading %d of %d characters", i, n),
				Wrapped: err,
			}
		}
	}
	rng.SetEnd(s.Offset())
	return rng, nil
}

func (s *Scanner) Range() Range {
	return Range{
		Start: s.Offset(),
		End:   s.Offset(),
		Path:  s.path,
		Text:  s.text,
	}
}

func Done[P interface {
	*T
	SetEnd(int)
}, T any](offset int, b P) T {
	b.SetEnd(offset)
	return *b
}
