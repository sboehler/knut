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
	// buffer contains the current backtracking buffer. Its size is always at
	// least one.
	buffer []rune
	// stops is a LIFO queue of indexes into buffer. The last element
	// is the most recent one.
	stops []int
	// bufpos is the current index into the buffer. When bufpos == len(bbuffer) -1,
	// the buffer is current.
	bufpos int
	// Position is the current position in the stream.
	Position int
	// Path is the file path.
	Path string
}

// New creates a new Scanner.
func New(r io.RuneReader, path string) (*Scanner, error) {
	b := &Scanner{
		reader:   r,
		buffer:   make([]rune, 1),
		Position: -1,
		Path:     path,
	}
	if err := b.Advance(); err != nil {
		return nil, err
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
	return s.buffer[s.bufpos]
}

// Advance reads a rune.
func (s *Scanner) Advance() error {
	if s.bufpos < len(s.buffer)-1 {
		s.bufpos++
		s.Position++
		return nil
	}
	// Invariant: s.pos == len(s.buffer) - 1
	ch, _, err := s.reader.ReadRune()
	if err == io.EOF {
		ch = EOF
		err = nil
	}
	if err != nil {
		return err
	}
	if len(s.stops) > 0 {
		s.buffer = append(s.buffer, ch)
		s.bufpos++
	} else {
		if s.bufpos > 0 {
			s.bufpos = 0
			s.buffer = s.buffer[:1]
		}
		s.buffer[s.bufpos] = ch
	}
	s.Position++
	return nil
}

// Begin begins a backtracking transaction
func (s *Scanner) Begin() {
	s.stops = append(s.stops, s.bufpos)
}

// Commit commits a backtracking transaction
func (s *Scanner) Commit() {
	if len(s.stops) > 0 {
		s.stops = s.stops[:len(s.stops)-1]
	}
}

// Rollback rolls back a backtracking transactin
func (s *Scanner) Rollback() {
	if len(s.stops) == 0 {
		return
	}
	s.Position -= (s.bufpos - s.stops[len(s.stops)-1])
	s.bufpos = s.stops[len(s.stops)-1]
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
