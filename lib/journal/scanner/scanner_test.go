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
