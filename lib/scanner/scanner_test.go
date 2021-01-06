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
	"strings"
	"testing"
)

func TestNewScanner(t *testing.T) {
	r, err := New(strings.NewReader(""), "")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if c := r.Current(); c != EOF {
		t.Fatalf("Expected EOF, got %c", c)
	}
}

func TestWithoutBacktracking(t *testing.T) {
	s := "foobar"
	b, err := New(strings.NewReader(s), "")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range s {
		if d := b.Current(); d != c {
			t.Fatalf("Expected %c, got %c", c, d)
		}
		b.Advance()
	}
	if c := b.Current(); c != EOF {
		t.Fatalf("Expected EOF, got %c", c)
	}
}

func TestBacktracking(t *testing.T) {
	s := "foobar"
	b, err := New(strings.NewReader(s), "")
	if err != nil {
		t.Fatal(err)
	}
	if b.Position != 0 {
		t.Errorf("b.Position = %d, want 0", b.Position)
	}
	b.Begin()
	b.Begin()
	for i := 0; i < 3; i++ {
		b.Advance()
	}
	if b.Position != 3 {
		t.Errorf("b.Position = %d, want 3", b.Position)
	}
	b.Commit()
	if b.Position != 3 {
		t.Errorf("b.Position = %d, want 3", b.Position)
	}
	b.Rollback()
	if b.Position != 0 {
		t.Errorf("b.Position = %d, want 0", b.Position)
	}
	for _, c := range s {
		if d := b.Current(); d != c {
			t.Fatalf("Expected %c, got %c", c, d)
		}
		b.Advance()
	}
	b.Commit()
	if b.Position != 6 {
		t.Errorf("b.Position = %d, want 6", b.Position)
	}
	if c := b.Current(); c != EOF {
		t.Fatalf("Expected EOF, got %c", c)
	}
}

func TestBacktrackingAndCommitWithShorterString(t *testing.T) {
	s := "foobar"
	b, err := New(strings.NewReader(s), "")
	if err != nil {
		t.Fatal(err)
	}
	b.Begin()
	for i := 0; i < 5; i++ {
		b.Advance()
	}
	b.Rollback()
	if b.Position != 0 {
		t.Errorf("b.Position = %d, want 0", b.Position)
	}
	for _, c := range "foo" {
		if d := b.Current(); d != c {
			t.Fatalf("Expected %c, got %c", c, d)
		}
		b.Advance()
	}
	b.Commit()
	if b.Position != 3 {
		t.Errorf("b.Position = %d, want 3", b.Position)
	}
	for _, c := range "bar" {
		if d := b.Current(); d != c {
			t.Fatalf("Expected %c, got %c", c, d)
		}
		b.Advance()
	}
	if b.bufpos != 0 {
		t.Fatalf("Expected pos=0, got %v", b.bufpos)
	}
	if len(b.buffer) != 1 {
		t.Fatalf("Expected len(buffer)=0, got %v", len(b.buffer))
	}
}
