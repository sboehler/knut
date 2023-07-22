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
	"testing"
	"unicode"
)

func TestNewScanner(t *testing.T) {
	s := New("", "")
	if err := s.Advance(); err != nil {
		t.Fatalf("s.Advance() = %#v, want nil", err)
	}
	if c := s.Current(); c != EOF {
		t.Fatalf("s.Current() = %c, want EOF", c)
	}
}

func TestReadN(t *testing.T) {
	for _, test := range []struct {
		n       int
		want    Range
		wantErr bool
	}{
		{
			n:    3,
			want: Range{0, 3},
		},
		{
			n:    6,
			want: Range{0, 6},
		},
		{
			n:       7,
			want:    Range{0, 6},
			wantErr: true,
		},
	} {
		t.Run(fmt.Sprintf("n=%d", test.n), func(t *testing.T) {
			scanner := setupScanner(t, "foobar")

			got, err := scanner.ReadN(test.n)

			if (err != nil) != test.wantErr {
				t.Fatalf("scanner.ReadN(%d) returned error %#v, want error presence %t", test.n, err, test.wantErr)
			}
			if got != test.want {
				t.Fatalf("scanner.ReadN(%d) = %v, %v, want %v, nil", test.n, got, err, test.want)
			}
		})
	}

}

func TestReadString(t *testing.T) {
	for _, test := range []struct {
		text    string
		str     string
		want    Range
		wantErr bool
	}{
		{
			str:  "",
			want: Range{0, 0},
		},
		{
			str:  "foo",
			want: Range{0, 3},
		},
		{
			str:  "foobar",
			want: Range{0, 6},
		},
		{
			str:     "foobarbaz",
			want:    Range{0, 6},
			wantErr: true,
		},
	} {
		t.Run(test.str, func(t *testing.T) {
			scanner := setupScanner(t, "foobar")

			got, err := scanner.ReadString(test.str)

			if (err != nil) != test.wantErr {
				t.Fatalf("scanner.ReadString(%s) returned error %#v, want error presence %t", test.str, err, test.wantErr)
			}
			if got != test.want {
				t.Fatalf("scanner.ReadString(%s) = %v, %v, want %v, nil", test.str, got, err, test.want)
			}
		})
	}
}

func TestReadCharacter(t *testing.T) {
	for _, test := range []struct {
		text    string
		char    rune
		want    Range
		wantErr bool
	}{
		{
			text: "foo",
			char: 'f',
			want: Range{0, 1},
		},
		{
			text:    "foo",
			char:    'o',
			want:    Range{0, 0},
			wantErr: true,
		},
		{
			text:    "",
			char:    'o',
			want:    Range{0, 0},
			wantErr: true,
		},
	} {
		t.Run(fmt.Sprintf("ReadChar %c in %s", test.char, test.text), func(t *testing.T) {
			scanner := setupScanner(t, "foobar")

			got, err := scanner.ReadCharacter(test.char)

			if (err != nil) != test.wantErr {
				t.Fatalf("scanner.ReadChar(%c) returned error %#v, want error presence %t", test.char, err, test.wantErr)
			}
			if got != test.want {
				t.Fatalf("scanner.ReadChar(%c) = %v, %v, want %v, nil", test.char, got, err, test.want)
			}
		})
	}
}

func TestReadWhile(t *testing.T) {
	for _, test := range []struct {
		text string
		pred func(rune) bool
		want Range
	}{
		{
			text: "ooobar",
			pred: func(r rune) bool { return r == 'o' },
			want: Range{0, 3},
		},
		{
			text: "ASDFasdf",
			pred: unicode.IsUpper,
			want: Range{0, 4},
		},
		{
			text: "ASDF",
			pred: unicode.IsUpper,
			want: Range{0, 4},
		},
	} {
		t.Run(test.text, func(t *testing.T) {
			scanner := setupScanner(t, test.text)

			got, err := scanner.ReadWhile(test.pred)

			if err != nil || got != test.want {
				t.Fatalf("scanner.ReadWhile(pred) = %v, %v, want %v, nil", got, err, test.want)
			}
		})
	}
}

func TestReadUntil(t *testing.T) {
	for _, test := range []struct {
		char    rune
		want    Range
		wantErr bool
	}{
		{
			char: 'r',
			want: Range{0, 5},
		},
		{
			char: 'f',
			want: Range{0, 0},
		},
		{
			char:    'z',
			want:    Range{0, 6},
			wantErr: true,
		},
	} {
		t.Run(string(test.char), func(t *testing.T) {
			scanner := setupScanner(t, "foobar")

			got, err := scanner.ReadUntil(func(r rune) bool { return r == test.char })

			if (err != nil) != test.wantErr {
				t.Fatalf("scanner.ReadUntil(pred) returned error %#v, want error presence %t", err, test.wantErr)
			}
			if got != test.want {
				t.Fatalf("scanner.ReadUntil(pred) = %v, %v, want %v, nil", got, err, test.want)
			}
		})
	}
}

func TestAdvanceAndCurrent(t *testing.T) {
	scanner := setupScanner(t, "foobar")
	for _, want := range "foobar" {

		got := scanner.Current()

		if want != got {
			t.Fatalf("s.Current() = %c, want %c", got, want)
		}
		if err := scanner.Advance(); err != nil {
			t.Fatalf("s.Advance() = %v, want nil", err)
		}
	}
	if got := scanner.Current(); got != EOF {
		t.Fatalf("s.Current() = %c want EOF", got)
	}
}

func setupScanner(t *testing.T, text string) *Scanner {
	t.Helper()
	scanner := New(text, "")
	if err := scanner.Advance(); err != nil {
		t.Fatalf("s.Advance() = %v, want nil", err)
	}
	return scanner
}
