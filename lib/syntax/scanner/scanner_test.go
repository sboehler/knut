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
	"testing"
	"unicode"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/sboehler/knut/lib/syntax/directives"
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
		n    int
		want Range
		err  error
	}{
		{
			n:    3,
			want: Range{Start: 0, End: 3, Text: "foobar"},
		},
		{
			n:    6,
			want: Range{Start: 0, End: 6, Text: "foobar"},
		},
		{
			n:    7,
			want: Range{Start: 0, End: 6, Text: "foobar"},
			err: directives.Error{
				Range:   directives.Range{End: 6, Text: "foobar"},
				Message: "while reading 6 of 7 characters",
				Wrapped: io.EOF,
			},
		},
	} {
		t.Run(fmt.Sprintf("n=%d", test.n), func(t *testing.T) {
			scanner := setupScanner(t, "foobar")

			got, err := scanner.ReadN(test.n)

			assert(t, fmt.Sprintf("scanner.ReadN(%d)", test.n), test.want, got, test.err, err)
		})
	}
}

func assert(t *testing.T, function, want, got any, wantErr, gotErr error) {
	t.Helper()
	if diff := cmp.Diff(wantErr, gotErr, cmpopts.EquateErrors()); diff != "" {
		t.Fatalf("%s returned unexpected diff in err (-want/+got)\n%s\n", function, diff)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("%s returned unexpected diff (-want/+got)\n%s\n", function, diff)
	}
}

func TestReadString(t *testing.T) {
	for _, test := range []struct {
		text    string
		str     string
		want    Range
		wantErr error
	}{
		{
			str:  "",
			want: Range{Start: 0, End: 0, Text: "foobar"},
		},
		{
			str:  "foo",
			want: Range{Start: 0, End: 3, Text: "foobar"},
		},
		{
			str:  "foobar",
			want: Range{Start: 0, End: 6, Text: "foobar"},
		},
		{
			str:  "foobarbaz",
			want: Range{Start: 0, End: 6, Text: "foobar"},
			wantErr: directives.Error{
				Message: "while reading \"foobarbaz\"",
				Range:   directives.Range{End: 6, Text: "foobar"},
			},
		},
	} {
		t.Run(test.str, func(t *testing.T) {
			scanner := setupScanner(t, "foobar")

			got, err := scanner.ReadString(test.str)

			assert(t, fmt.Sprintf("scanner.ReadString(%s)", test.str), test.want, got, test.wantErr, err)
		})
	}
}

func TestReadCharacter(t *testing.T) {
	for _, test := range []struct {
		text    string
		char    rune
		want    Range
		wantErr error
	}{
		{
			text: "foobar",
			char: 'f',
			want: Range{Start: 0, End: 1, Text: "foobar"},
		},
		{
			text: "foo",
			char: 'o',
			want: Range{Start: 0, End: 0, Text: "foo"},
			wantErr: directives.Error{
				Message: "unexpected character `f`, want `o`",
				Range:   Range{Start: 0, End: 0, Text: "foo"},
			},
		},
		{
			text: "",
			char: 'o',
			want: Range{Start: 0, End: 0, Text: ""},
			wantErr: directives.Error{
				Message: "unexpected end of file, want `o`",
				Range:   Range{Start: 0, End: 0, Text: ""},
			},
		},
	} {
		t.Run(fmt.Sprintf("ReadChar %c in %s", test.char, test.text), func(t *testing.T) {
			scanner := setupScanner(t, test.text)

			got, err := scanner.ReadCharacter(test.char)

			assert(t, fmt.Sprintf("scanner.ReadCharacter(%c)", test.char), test.want, got, test.wantErr, err)
		})
	}
}

func TestReadCharacterWith(t *testing.T) {
	for _, test := range []struct {
		text    string
		char    rune
		want    Range
		wantErr error
	}{
		{
			text: "foo",
			char: 'f',
			want: Range{Start: 0, End: 1, Text: "foo"},
		},
		{
			text: "foo",
			char: 'o',
			want: Range{Start: 0, End: 0, Text: "foo"},
			wantErr: directives.Error{
				Message: "unexpected character `f`, want character `o`",
				Range:   Range{Text: "foo"},
			},
		},
		{
			text: "",
			char: 'o',
			want: Range{Start: 0, End: 0, Text: ""},
			wantErr: directives.Error{
				Message: "unexpected end of file, want character `o`",
				Range:   Range{Start: 0, End: 0, Text: ""},
			},
		},
	} {
		t.Run(fmt.Sprintf("ReadChar %c in %s", test.char, test.text), func(t *testing.T) {
			scanner := setupScanner(t, test.text)
			desc := fmt.Sprintf("character `%c`", test.char)

			got, err := scanner.ReadCharacterWith(desc, func(r rune) bool { return r == test.char })

			assert(t, fmt.Sprintf("scanner.ReadCharacterWith(%s)", desc), test.want, got, test.wantErr, err)
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
			want: Range{Start: 0, End: 3, Text: "ooobar"},
		},
		{
			text: "",
			pred: func(r rune) bool { return r == 'o' },
			want: Range{Start: 0, End: 0, Text: ""},
		},
		{
			text: "ASDFasdf",
			pred: unicode.IsUpper,
			want: Range{Start: 0, End: 4, Text: "ASDFasdf"},
		},
		{
			text: "ASDF",
			pred: unicode.IsUpper,
			want: Range{Start: 0, End: 4, Text: "ASDF"},
		},
	} {
		t.Run(test.text, func(t *testing.T) {
			scanner := setupScanner(t, test.text)

			got, err := scanner.ReadWhile(test.pred)

			assert(t, "scanner.ReadWhile()", test.want, got, nil, err)

		})
	}
}

func TestReadWhile1(t *testing.T) {
	for _, test := range []struct {
		text    string
		pred    func(rune) bool
		desc    string
		want    Range
		wantErr error
	}{
		{
			text: "ooobar",
			pred: func(r rune) bool { return r == 'o' },
			desc: "character `o`",
			want: Range{Start: 0, End: 3, Text: "ooobar"},
		},
		{
			text: "",
			pred: func(r rune) bool { return r == 'o' },
			want: Range{Start: 0, End: 0, Text: ""},
			desc: "character `o`",
			wantErr: directives.Error{
				Message: "unexpected end of file, want character `o`",
				Range:   Range{},
			},
		},
		{
			text: "ASDFasdf",
			pred: unicode.IsUpper,
			desc: "an upper-case character",
			want: Range{Start: 0, End: 4, Text: "ASDFasdf"},
		},
		{
			text: "ASDF",
			pred: unicode.IsUpper,
			desc: "an upper-case character",
			want: Range{Start: 0, End: 4, Text: "ASDF"},
		},
		{
			text: "asdf",
			pred: unicode.IsUpper,
			desc: "an upper-case character",
			want: Range{Start: 0, End: 0, Text: "asdf"},
			wantErr: directives.Error{
				Message: "unexpected character `a`, want an upper-case character",
				Range:   Range{Start: 0, End: 0, Text: "asdf"},
			},
		},
	} {
		t.Run(test.text, func(t *testing.T) {
			scanner := setupScanner(t, test.text)

			got, err := scanner.ReadWhile1(test.desc, test.pred)

			assert(t, fmt.Sprintf("scanner.ReadWhile1(%s)", test.desc), test.want, got, test.wantErr, err)
		})
	}
}

func TestReadUntil(t *testing.T) {
	for _, test := range []struct {
		char    rune
		want    Range
		wantErr error
	}{
		{
			char: 'r',
			want: Range{Start: 0, End: 5, Text: "foobar"},
		},
		{
			char: 'f',
			want: Range{Start: 0, End: 0, Text: "foobar"},
		},
		{
			char: 'z',
			want: Range{Start: 0, End: 6, Text: "foobar"},
			wantErr: directives.Error{
				Message: "unexpected end of file, want character `z`",
				Range:   Range{Start: 0, End: 6, Text: "foobar"},
			},
		},
	} {
		t.Run(string(test.char), func(t *testing.T) {
			scanner := setupScanner(t, "foobar")
			desc := fmt.Sprintf("character `%c`", test.char)

			got, err := scanner.ReadUntil(desc, func(r rune) bool { return r == test.char })

			assert(t, fmt.Sprintf("scanner.ReadUntil(%s)", desc), test.want, got, test.wantErr, err)
		})
	}
}

func TestReadAlternative(t *testing.T) {
	for _, test := range []struct {
		text    string
		input   []string
		want    Range
		wantErr error
	}{
		{
			text:  "foobar",
			input: []string{"foo1", "foo2", "foo"},
			want:  Range{End: 3, Text: "foobar"},
		},
		{
			text:  "foobar",
			input: []string{"baz", "bar", "foo"},
			want:  Range{End: 3, Text: "foobar"},
		},
		{
			text:  "",
			input: []string{"baz", "bar", "foo"},
			want:  Range{Text: ""},
			wantErr: directives.Error{
				Message: "unexpected end of file, want one of {`baz`, `bar`, `foo`}",
				Range:   Range{Text: ""},
			},
		},
		{
			text:  "foobar",
			input: []string{"baz", "bar"},
			want:  Range{Text: "foobar"},
			wantErr: directives.Error{
				Message: "unexpected input, want one of {`baz`, `bar`}",
				Range:   Range{Text: "foobar"},
			},
		},
	} {
		t.Run(fmt.Sprintf("%#v", test.input), func(t *testing.T) {
			scanner := setupScanner(t, test.text)

			got, err := scanner.ReadAlternative(test.input)

			assert(t, fmt.Sprintf("scanner.ReadAlternative(%#v)", test.input), test.want, got, test.wantErr, err)
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
