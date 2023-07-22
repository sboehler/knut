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
		wantEnd int
		wantErr bool
	}{
		{
			n:       3,
			wantEnd: 3,
		},
		{
			n:       6,
			wantEnd: 6,
		},
		{
			n:       7,
			wantEnd: 6,
			wantErr: true,
		},
	} {
		t.Run(fmt.Sprintf("n=%d", test.n), func(t *testing.T) {
			scanner := setupScanner(t, "foobar")

			gotStart, gotEnd, err := scanner.ReadN(test.n)

			if (err != nil) != test.wantErr {
				t.Fatalf("scanner.ReadN(%d) returned error %#v, want error presence %t", test.n, err, test.wantErr)
			}
			if gotStart != 0 || gotEnd != test.wantEnd {
				t.Fatalf("scanner.ReadN(%d) = %d, %d, %v, want %d, %d, nil", test.n, gotStart, gotEnd, err, 0, test.wantEnd)
			}
		})
	}

}

func TestReadString(t *testing.T) {
	for _, test := range []struct {
		text    string
		str     string
		wantEnd int
		wantErr bool
	}{
		{
			str:     "",
			wantEnd: 0,
		},
		{
			str:     "foo",
			wantEnd: 3,
		},
		{
			str:     "foobar",
			wantEnd: 6,
		},
		{
			str:     "foobarbaz",
			wantEnd: 6,
			wantErr: true,
		},
	} {
		t.Run(test.str, func(t *testing.T) {
			scanner := setupScanner(t, "foobar")

			gotStart, gotEnd, err := scanner.ReadString(test.str)

			if (err != nil) != test.wantErr {
				t.Fatalf("scanner.ReadString(%s) returned error %#v, want error presence %t", test.str, err, test.wantErr)
			}
			if gotStart != 0 || gotEnd != test.wantEnd {
				t.Fatalf("scanner.ReadString(%s) = %d, %d, %v, want %d, %d, nil", test.str, gotStart, gotEnd, err, 0, test.wantEnd)
			}
		})
	}
}

func TestReadCharacter(t *testing.T) {
	for _, test := range []struct {
		text    string
		char    rune
		wantEnd int
		wantErr bool
	}{
		{
			text:    "foo",
			char:    'f',
			wantEnd: 1,
		},
		{
			text:    "foo",
			char:    'o',
			wantEnd: 0,
			wantErr: true,
		},
		{
			text:    "",
			char:    'o',
			wantEnd: 0,
			wantErr: true,
		},
	} {
		t.Run(fmt.Sprintf("ReadChar %c in %s", test.char, test.text), func(t *testing.T) {
			scanner := setupScanner(t, "foobar")

			gotStart, gotEnd, err := scanner.ReadCharacter(test.char)

			if (err != nil) != test.wantErr {
				t.Fatalf("scanner.ReadChar(%c) returned error %#v, want error presence %t", test.char, err, test.wantErr)
			}
			if gotStart != 0 || gotEnd != test.wantEnd {
				t.Fatalf("scanner.ReadChar(%c) = %d, %d, %v, want %d, %d, nil", test.char, gotStart, gotEnd, err, 0, test.wantEnd)
			}
		})
	}
}

func TestReadWhile(t *testing.T) {
	for _, test := range []struct {
		text string
		pred func(rune) bool
		want int
	}{
		{
			text: "ooobar",
			pred: func(r rune) bool { return r == 'o' },
			want: 3,
		},
		{
			text: "ASDFasdf",
			pred: unicode.IsUpper,
			want: 4,
		},
		{
			text: "ASDF",
			pred: unicode.IsUpper,
			want: 4,
		},
	} {
		t.Run(test.text, func(t *testing.T) {
			scanner := setupScanner(t, test.text)

			gotStart, gotEnd, err := scanner.ReadWhile(test.pred)

			if err != nil || gotStart != 0 || gotEnd != test.want {
				t.Fatalf("scanner.ReadWhile(pred) = %d, %d, %v, want %d, %d, nil", gotStart, gotEnd, err, 0, test.want)
			}
		})
	}
}

func TestReadUntil(t *testing.T) {
	for _, test := range []struct {
		char    rune
		wantEnd int
		wantErr bool
	}{
		{
			char:    'r',
			wantEnd: 5,
		},
		{
			char:    'f',
			wantEnd: 0,
		},
		{
			char:    'z',
			wantEnd: 6,
			wantErr: true,
		},
	} {
		t.Run(string(test.char), func(t *testing.T) {
			scanner := setupScanner(t, "foobar")

			gotStart, gotEnd, err := scanner.ReadUntil(func(r rune) bool { return r == test.char })

			if (err != nil) != test.wantErr {
				t.Fatalf("scanner.ReadUntil(pred) returned error %#v, want error presence %t", err, test.wantErr)
			}
			if gotStart != 0 || gotEnd != test.wantEnd {
				t.Fatalf("scanner.ReadUntil(pred) = %d, %d, %v, want %d, %d, nil", gotStart, gotEnd, err, 0, test.wantEnd)
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
