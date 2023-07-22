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
			return Range{start, s.pos}, err
		}
	}
	return Range{start, s.pos}, nil
}

// ReadUntil advances the scanner until the predicate holds.
func (s *Scanner) ReadUntil(pred func(r rune) bool) (Range, error) {
	start := s.pos
	for !pred(s.Current()) {
		if err := s.Advance(); err != nil {
			return Range{start, s.pos}, err
		}
		if s.Current() == EOF {
			return Range{start, s.pos}, fmt.Errorf("unexpected end of file")
		}
	}
	return Range{start, s.pos}, nil
}

// ReadCharacter consumes the given rune.
func (s *Scanner) ReadCharacter(r rune) (Range, error) {
	if s.Current() != r {
		return Range{s.pos, s.pos}, fmt.Errorf("expected %c, got %c", r, s.Current())
	}
	start := s.pos
	err := s.Advance()
	return Range{start, s.pos}, err
}

// ReadString parses the given string.
func (s *Scanner) ReadString(str string) (Range, error) {
	start := s.pos
	for _, ch := range str {
		if ch != s.Current() {
			return Range{start, s.pos}, fmt.Errorf("expected %v, got %v", str, s.text[start:s.pos])
		}
		if err := s.Advance(); err != nil {
			return Range{start, s.pos}, err
		}
	}
	return Range{start, s.pos}, nil
}

// ReadN reads a string with n runes.
func (s *Scanner) ReadN(n int) (Range, error) {
	start := s.pos
	for i := 0; i < n; i++ {
		if s.current == EOF {
			return Range{start, s.pos}, io.EOF
		}
		if err := s.Advance(); err != nil {
			return Range{start, s.pos}, err
		}
	}
	return Range{start, s.pos}, nil
}

type Range struct {
	Start, End int
}
