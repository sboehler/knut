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
	if s.offset == len(s.text) {
		s.current = EOF
		s.currentLen = 0
		return nil
	}
	s.current, s.currentLen = utf8.DecodeRuneInString(s.text[s.offset:])
	if s.current == utf8.RuneError {
		switch s.currentLen {
		case 0:
			return fmt.Errorf("unexpected end of file: %s", s.text[s.offset:])
		case 1:
			return fmt.Errorf("invalid string: %s", s.text[s.offset:])
		}
	}
	return nil
}

// EOF is a rune representing the end of a file
const EOF = rune(0)

// ReadWhile reads a string while the predicate holds.
func (s *Scanner) ReadWhile(pred func(r rune) bool) (Range, error) {
	rng := s.Range()
	for pred(s.Current()) && s.Current() != EOF {
		if err := s.Advance(); err != nil {
			return updateRange(s, &rng), err
		}
	}
	return updateRange(s, &rng), nil
}

// ReadWhile reads a string while the predicate holds. The predicate must be
// satisfied at least once.
func (s *Scanner) ReadWhile1(pred func(r rune) bool) (Range, error) {
	rng := s.Range()
	if !pred(s.Current()) {
		return updateRange(s, &rng), fmt.Errorf("unexpected character %c", s.Current())
	}
	if s.Current() == EOF {
		return updateRange(s, &rng), fmt.Errorf("unexpected end of file")
	}
	for pred(s.Current()) && s.Current() != EOF {
		if err := s.Advance(); err != nil {
			return updateRange(s, &rng), err
		}
	}
	return updateRange(s, &rng), nil
}

// ReadUntil advances the scanner until the predicate holds.
func (s *Scanner) ReadUntil(pred func(r rune) bool) (Range, error) {
	rng := s.Range()
	for !pred(s.Current()) {
		if err := s.Advance(); err != nil {
			return updateRange(s, &rng), err
		}
		if s.Current() == EOF {
			return updateRange(s, &rng), fmt.Errorf("unexpected end of file")
		}
	}
	return updateRange(s, &rng), nil
}

// ReadCharacter consumes the given rune.
func (s *Scanner) ReadCharacter(r rune) (Range, error) {
	if s.Current() != r {
		return s.Range(), fmt.Errorf("expected %c, got %c", r, s.Current())
	}
	rng := s.Range()
	err := s.Advance()
	return updateRange(s, &rng), err
}

// ReadCharacter optionally consumes the given rune.
func (s *Scanner) ReadCharacterWith(pred func(rune) bool) (Range, error) {
	rng := s.Range()
	if !pred(s.Current()) {
		return updateRange(s, &rng), fmt.Errorf("unexpected character: %c", s.Current())
	}
	err := s.Advance()
	return updateRange(s, &rng), err
}

// ReadString parses the given string.
func (s *Scanner) ReadString(str string) (Range, error) {
	rng := s.Range()
	for _, ch := range str {
		if ch != s.Current() {
			return updateRange(s, &rng), fmt.Errorf("expected %v, got %v", str, s.text[rng.Start:s.Offset()])
		}
		if err := s.Advance(); err != nil {
			return updateRange(s, &rng), err
		}
	}
	return updateRange(s, &rng), nil
}

// ReadN reads a string with n runes.
func (s *Scanner) ReadN(n int) (Range, error) {
	rng := s.Range()
	for i := 0; i < n; i++ {
		if s.current == EOF {
			return updateRange(s, &rng), io.EOF
		}
		if err := s.Advance(); err != nil {
			return updateRange(s, &rng), err
		}
	}
	return updateRange(s, &rng), nil
}

func (s *Scanner) Range() Range {
	return Range{
		Start: s.Offset(),
		End:   s.Offset(),
		Path:  s.path,
		Text:  s.text,
	}
}

func updateRange[P interface {
	*T
	SetEnd(int)
}, T any](p *Scanner, b P) T {
	b.SetEnd(p.Offset())
	return *b
}
