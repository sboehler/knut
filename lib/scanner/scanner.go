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
	"io"
	"strings"
	"unicode/utf8"
)

// Scanner is a backtracking reader.
type Scanner struct {
	reader io.RuneReader
	// current contains the current rune
	current rune
	// Position is the current position in the stream.
	Position int
	// Path is the file path.
	Path string
	// Position2 is the current position in the stream.
	pos Position
}

// Position is a position of a character in a text file.
type Position struct {
	Path                           string
	BytePos, RunePos, Line, Column int
}

// New creates a new Scanner.
func New(r io.RuneReader, path string) (*Scanner, error) {
	ch, _, err := r.ReadRune()
	if err != nil {
		if err != io.EOF {
			return nil, err
		}
		ch = EOF
	}
	b := &Scanner{
		reader:   r,
		current:  ch,
		Position: 0,
		Path:     path,
		pos: Position{
			Path: path,
		},
	}
	return b, nil
}

// ReadRune implements io.RuneReader.
func (s *Scanner) ReadRune() (r rune, size int, err error) {
	if err := s.Advance(); err != nil {
		return 0, 0, err
	}
	return s.Current(), utf8.RuneLen(s.Current()), nil
}

// Current returns the current rune.
func (s *Scanner) Current() rune {
	return s.current
}

// Advance reads a rune.
func (s *Scanner) Advance() error {
	ch, _, err := s.reader.ReadRune()
	if err != nil {
		if err != io.EOF {
			return err
		}
		ch = EOF
	}
	s.pos.BytePos += utf8.RuneLen(s.current)
	s.pos.RunePos++
	if s.current == '\n' {
		s.pos.Line++
		s.pos.Column = 0
	} else {
		s.pos.Column++
	}
	s.current = ch
	return nil
}

// EOF is a rune representing the end of a file
const EOF = rune(0)

// ReadWhile reads runes into the builder while the predicate holds
func (s *Scanner) ReadWhile(pred func(r rune) bool) (string, error) {
	b := strings.Builder{}
	for pred(s.Current()) && s.Current() != EOF {
		b.WriteRune(s.Current())
		if err := s.Advance(); err != nil {
			return b.String(), err
		}
	}
	return b.String(), nil
}

// ConsumeWhile advances the parser while the predicate holds
func (s *Scanner) ConsumeWhile(pred func(r rune) bool) error {
	for pred(s.Current()) {
		if err := s.Advance(); err != nil {
			return err
		}
	}
	return nil
}

// ConsumeUntil advances the parser until the predicate holds
func (s *Scanner) ConsumeUntil(pred func(r rune) bool) error {
	for !pred(s.Current()) {
		if err := s.Advance(); err != nil {
			return err
		}
	}
	return nil
}

// ConsumeRune consumes the given rune
func (s *Scanner) ConsumeRune(r rune) error {
	if s.Current() != r {
		return fmt.Errorf("Expected %c, got %c", r, s.Current())
	}
	return s.Advance()
}

// ParseString parses the given string
func (s *Scanner) ParseString(str string) error {
	b := strings.Builder{}
	for _, ch := range str {
		if _, err := b.WriteRune(s.Current()); err != nil {
			return err
		}
		if ch != s.Current() {
			return fmt.Errorf("Expected %v, got %v", str, b.String())
		}
		if err := s.Advance(); err != nil {
			return err
		}
	}
	return nil
}
