package parser

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/sboehler/knut/lib/syntax"
)

func TestParseCommodity(t *testing.T) {
	for _, test := range []struct {
		text    string
		want    syntax.Commodity
		wantErr bool
	}{
		{
			text: "foobar",
			want: syntax.Commodity{Start: 0, End: 6},
		},
		{
			text:    "",
			want:    syntax.Commodity{Start: 0, End: 0},
			wantErr: true,
		},
		{
			text:    "(foobar)",
			want:    syntax.Commodity{Start: 0, End: 0},
			wantErr: true,
		},
	} {
		t.Run(test.text, func(t *testing.T) {
			p := setupParser(t, test.text)

			got, err := p.parseCommodity()

			if (err != nil) != test.wantErr || !cmp.Equal(got, test.want, cmpopts.IgnoreFields(syntax.Commodity{}, "Text")) {
				t.Fatalf("p.parseCommodity() = %#v, %#v, want %#v, error presence %t", got, err, test.want, test.wantErr)
			}
		})
	}
}

func TestParseAccount(t *testing.T) {
	for _, test := range []struct {
		text    string
		want    syntax.Account
		wantErr bool
	}{
		{
			text: "foobar",
			want: syntax.Account{Start: 0, End: 6},
		},
		{
			text:    "",
			want:    syntax.Account{Start: 0, End: 0},
			wantErr: true,
		},
		{
			text:    "(foobar)",
			want:    syntax.Account{Start: 0, End: 0},
			wantErr: true,
		},
		{
			text:    "ABC:",
			want:    syntax.Account{Start: 0, End: 4},
			wantErr: true,
		},
		{
			text: "ABC:B",
			want: syntax.Account{Start: 0, End: 5},
		},
		{
			text: "ABC:B:C:D",
			want: syntax.Account{Start: 0, End: 9},
		},
	} {
		t.Run(test.text, func(t *testing.T) {
			p := setupParser(t, test.text)

			got, err := p.parseAccount()

			if (err != nil) != test.wantErr || !cmp.Equal(got, test.want, cmpopts.IgnoreFields(syntax.Account{}, "Text")) {
				t.Fatalf("p.parseAccount() = %#v, %#v, want %#v, error presence %t", got, err, test.want, test.wantErr)
			}
		})
	}
}

func TestParseAccountMacro(t *testing.T) {
	for _, test := range []struct {
		text    string
		want    syntax.AccountMacro
		wantErr bool
	}{
		{
			text: "$foobar",
			want: syntax.AccountMacro{Start: 0, End: 7},
		},
		{
			text: "$foo1",
			want: syntax.AccountMacro{Start: 0, End: 4},
		},
		{
			text:    "$1foo",
			want:    syntax.AccountMacro{Start: 0, End: 1},
			wantErr: true,
		},
		{
			text:    "",
			want:    syntax.AccountMacro{Start: 0, End: 0},
			wantErr: true,
		},
		{
			text:    "foobar",
			want:    syntax.AccountMacro{Start: 0, End: 0},
			wantErr: true,
		},
	} {
		t.Run(test.text, func(t *testing.T) {
			p := setupParser(t, test.text)

			got, err := p.parseAccountMacro()

			if (err != nil) != test.wantErr || !cmp.Equal(got, test.want, cmpopts.IgnoreFields(syntax.AccountMacro{}, "Text")) {
				t.Fatalf("p.parseAccountMacro() = %#v, %#v, want %#v, error presence %t", got, err, test.want, test.wantErr)
			}
		})
	}
}

func TestParseDecimal(t *testing.T) {
	for _, test := range []struct {
		text    string
		want    syntax.Decimal
		wantErr bool
	}{
		{
			text: "10",
			want: syntax.Decimal{Start: 0, End: 2},
		},
		{
			text: "-10",
			want: syntax.Decimal{Start: 0, End: 3},
		},
		{
			text: "-10.0",
			want: syntax.Decimal{Start: 0, End: 5},
		},
		{
			text:    "-10.",
			want:    syntax.Decimal{Start: 0, End: 4},
			wantErr: true,
		},
		{
			text:    "99.",
			want:    syntax.Decimal{Start: 0, End: 3},
			wantErr: true,
		},
		{
			text:    "foo",
			want:    syntax.Decimal{Start: 0, End: 0},
			wantErr: true,
		},
	} {
		t.Run(test.text, func(t *testing.T) {
			p := setupParser(t, test.text)

			got, err := p.parseDecimal()

			if (err != nil) != test.wantErr || !cmp.Equal(got, test.want, cmpopts.IgnoreFields(syntax.Decimal{}, "Text")) {
				t.Fatalf("p.parseDecimal() = %#v, %#v, want %#v, error presence %t", got, err, test.want, test.wantErr)
			}
		})
	}
}

func setupParser(t *testing.T, text string) *Parser {
	t.Helper()
	parser := New(text, "")
	if err := parser.Advance(); err != nil {
		t.Fatalf("s.Advance() = %v, want nil", err)
	}
	return parser
}
