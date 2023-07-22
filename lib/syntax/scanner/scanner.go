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
)

// Scanner is a scanner.
type Scanner struct {
	text string
	path string

	// current contains the current rune
	current    rune
	currentLen int
	pos        int
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
	return s.pos
}

// Advance reads a rune.
func (s *Scanner) Advance() error {
	s.pos += s.currentLen
	if s.pos == len(s.text) {
		s.current = EOF
		s.currentLen = 0
		return nil
	}
	s.current, s.currentLen = utf8.DecodeRuneInString(s.text[s.pos:])
	if s.current == utf8.RuneError {
		switch s.currentLen {
		case 0:
			return fmt.Errorf("unexpected end of file: %s", s.text[s.pos:])
		case 1:
			return fmt.Errorf("invalid string: %s", s.text[s.pos:])
		}
	}
	return nil
}

// EOF is a rune representing the end of a file
const EOF = rune(0)

// ReadWhile reads a string while the predicate holds.
func (s *Scanner) ReadWhile(pred func(r rune) bool) (Range, error) {
	start := s.pos
	for pred(s.Current()) && s.Current() != EOF {
		if err := s.Advance(); err != nil {
			return s.Range(start), err
		}
	}
	return s.Range(start), nil
}

// ReadWhile reads a string while the predicate holds. The predicate must be
// satisfied at least once.
func (s *Scanner) ReadWhile1(pred func(r rune) bool) (Range, error) {
	start := s.pos
	if !pred(s.Current()) {
		return s.Range(start), fmt.Errorf("unexpected character %c", s.Current())
	}
	if s.Current() == EOF {
		return s.Range(start), fmt.Errorf("unexpected end of file")
	}
	for pred(s.Current()) && s.Current() != EOF {
		if err := s.Advance(); err != nil {
			return s.Range(start), err
		}
	}
	return s.Range(start), nil
}

// ReadUntil advances the scanner until the predicate holds.
func (s *Scanner) ReadUntil(pred func(r rune) bool) (Range, error) {
	start := s.pos
	for !pred(s.Current()) {
		if err := s.Advance(); err != nil {
			return s.Range(start), err
		}
		if s.Current() == EOF {
			return s.Range(start), fmt.Errorf("unexpected end of file")
		}
	}
	return s.Range(start), nil
}

// ReadCharacter consumes the given rune.
func (s *Scanner) ReadCharacter(r rune) (Range, error) {
	if s.Current() != r {
		return s.Range(s.pos), fmt.Errorf("expected %c, got %c", r, s.Current())
	}
	start := s.pos
	err := s.Advance()
	return s.Range(start), err
}

// ReadCharacter optionally consumes the given rune.
func (s *Scanner) ReadCharacterOpt(r rune) (Range, error) {
	if s.Current() != r {
		return s.Range(s.pos), nil
	}
	start := s.pos
	err := s.Advance()
	return s.Range(start), err
}

// ReadString parses the given string.
func (s *Scanner) ReadString(str string) (Range, error) {
	start := s.pos
	for _, ch := range str {
		if ch != s.Current() {
			return s.Range(start), fmt.Errorf("expected %v, got %v", str, s.text[start:s.pos])
		}
		if err := s.Advance(); err != nil {
			return s.Range(start), err
		}
	}
	return s.Range(start), nil
}

// ReadN reads a string with n runes.
func (s *Scanner) ReadN(n int) (Range, error) {
	start := s.pos
	for i := 0; i < n; i++ {
		if s.current == EOF {
			return s.Range(start), io.EOF
		}
		if err := s.Advance(); err != nil {
			return s.Range(start), err
		}
	}
	return s.Range(start), nil
}

func (s *Scanner) Range(start int) Range {
	return Range{
		Start: start,
		End:   s.Offset(),
		Path:  s.path,
		Text:  s.text,
	}
}

type Range struct {
	Start, End int
	Path, Text string
}
