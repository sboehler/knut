package parser

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sboehler/knut/lib/syntax"
)

type testcase[T any] struct {
	text    string
	want    func(string) T
	wantErr bool
}

type parserTest[T any] struct {
	tests []testcase[T]
	desc  string
	fn    func(p *Parser) (T, error)
}

func (tests parserTest[T]) run(t *testing.T) {
	t.Helper()
	for _, test := range tests.tests {
		t.Run(test.text, func(t *testing.T) {
			parser := New(test.text, "")
			if err := parser.Advance(); err != nil {
				t.Fatalf("s.Advance() = %v, want nil", err)
			}

			got, err := tests.fn(parser)

			if (err != nil) != test.wantErr {
				t.Fatalf("%s returned error %v, want error presence %t", tests.desc, err, test.wantErr)
			}
			if diff := cmp.Diff(test.want(test.text), got); diff != "" {
				t.Fatalf("%s returned unexpected diff (-want/+got)\n%s\n", tests.desc, diff)
			}
		})
	}
}

func TestParseCommodity(t *testing.T) {
	parserTest[syntax.Commodity]{
		tests: []testcase[syntax.Commodity]{
			{
				text: "foobar",
				want: func(s string) syntax.Commodity {
					return syntax.Commodity{Start: 0, End: 6, Text: s}
				},
			},
			{
				text: "",
				want: func(s string) syntax.Commodity {
					return syntax.Commodity{Start: 0, End: 0, Text: s}
				},
				wantErr: true,
			},
			{
				text: "(foobar)",
				want: func(s string) syntax.Commodity {
					return syntax.Commodity{Start: 0, End: 0, Text: s}
				},
				wantErr: true,
			},
		},
		fn: func(p *Parser) (syntax.Commodity, error) {
			return p.parseCommodity()
		},
		desc: "p.parseCommodity()",
	}.run(t)
}

func TestParseAccount(t *testing.T) {
	parserTest[syntax.Account]{
		tests: []testcase[syntax.Account]{
			{
				text: "foobar",
				want: func(s string) syntax.Account {
					return syntax.Account{Start: 0, End: 6, Text: s}
				},
			},
			{
				text: "",
				want: func(s string) syntax.Account {
					return syntax.Account{Start: 0, End: 0, Text: s}
				},
				wantErr: true,
			},
			{
				text: "(foobar)",
				want: func(s string) syntax.Account {
					return syntax.Account{Start: 0, End: 0, Text: s}
				},
				wantErr: true,
			},
			{
				text: "ABC:",
				want: func(s string) syntax.Account {
					return syntax.Account{Start: 0, End: 4, Text: s}
				},
				wantErr: true,
			},
			{
				text: "ABC:B",
				want: func(s string) syntax.Account {
					return syntax.Account{Start: 0, End: 5, Text: s}
				},
			},
			{
				text: "ABC:B:C:D",
				want: func(s string) syntax.Account {
					return syntax.Account{Start: 0, End: 9, Text: s}
				},
			},
		},
		desc: "p.parseAccount()",
		fn: func(p *Parser) (syntax.Account, error) {
			return p.parseAccount()
		},
	}.run(t)
}

func TestParseAccountMacro(t *testing.T) {
	parserTest[syntax.AccountMacro]{
		tests: []testcase[syntax.AccountMacro]{
			{
				text: "$foobar",
				want: func(s string) syntax.AccountMacro {
					return syntax.AccountMacro{Start: 0, End: 7, Text: s}
				},
			},
			{
				text: "$foo1",
				want: func(s string) syntax.AccountMacro {
					return syntax.AccountMacro{Start: 0, End: 4, Text: s}
				},
			},
			{
				text: "$1foo",
				want: func(s string) syntax.AccountMacro {
					return syntax.AccountMacro{Start: 0, End: 1, Text: s}
				},
				wantErr: true,
			},
			{
				text: "",
				want: func(s string) syntax.AccountMacro {
					return syntax.AccountMacro{Start: 0, End: 0, Text: s}
				},
				wantErr: true,
			},
			{
				text: "foobar",
				want: func(s string) syntax.AccountMacro {
					return syntax.AccountMacro{Start: 0, End: 0, Text: s}
				},
				wantErr: true,
			},
		},
		desc: "p.parseAccountMacro()",
		fn: func(p *Parser) (syntax.AccountMacro, error) {
			return p.parseAccountMacro()
		},
	}.run(t)
}

func TestParseDecimal(t *testing.T) {
	parserTest[syntax.Decimal]{
		tests: []testcase[syntax.Decimal]{
			{
				text: "10",
				want: func(s string) syntax.Decimal {
					return syntax.Decimal{Start: 0, End: 2, Text: s}
				},
			},
			{
				text: "-10",
				want: func(s string) syntax.Decimal {
					return syntax.Decimal{Start: 0, End: 3, Text: s}
				},
			},
			{
				text: "-10.0",
				want: func(s string) syntax.Decimal {
					return syntax.Decimal{Start: 0, End: 5, Text: s}
				},
			},
			{
				text: "-10.",
				want: func(s string) syntax.Decimal {
					return syntax.Decimal{Start: 0, End: 4, Text: s}
				},
				wantErr: true,
			},
			{
				text: "99.",
				want: func(s string) syntax.Decimal {
					return syntax.Decimal{Start: 0, End: 3, Text: s}
				},
				wantErr: true,
			},
			{
				text: "foo",
				want: func(s string) syntax.Decimal {
					return syntax.Decimal{Start: 0, End: 0, Text: s}
				},
				wantErr: true,
			},
		},
		desc: "p.parseDecimal()",
		fn: func(p *Parser) (syntax.Decimal, error) {
			return p.parseDecimal()
		},
	}.run(t)
}

func TestParseDate(t *testing.T) {
	parserTest[syntax.Date]{
		tests: []testcase[syntax.Date]{
			{
				text: "2023-05-31",
				want: func(s string) syntax.Date {
					return syntax.Date{Start: 0, End: 10, Text: s}
				},
			},
			{
				text: "202-05-31",
				want: func(s string) syntax.Date {
					return syntax.Date{Start: 0, End: 3, Text: s}
				},
				wantErr: true,
			},
			{
				text: "20205-31",
				want: func(s string) syntax.Date {
					return syntax.Date{Start: 0, End: 4, Text: s}
				},
				wantErr: true,
			},
		},
		desc: "p.parseDate()",
		fn: func(p *Parser) (syntax.Date, error) {
			return p.parseDate()
		},
	}.run(t)
}

func TestParseBooking(t *testing.T) {
	parserTest[syntax.Booking]{
		tests: []testcase[syntax.Booking]{
			{
				text: "A:B C:D 100.0 CHF",
				want: func(t string) syntax.Booking {
					return syntax.Booking{
						Pos:       syntax.Pos{Start: 0, End: 17, Text: t},
						Credit:    syntax.Account{Start: 0, End: 3, Text: t},
						Debit:     syntax.Account{Start: 4, End: 7, Text: t},
						Amount:    syntax.Decimal{Start: 8, End: 13, Text: t},
						Commodity: syntax.Commodity{Start: 14, End: 17, Text: t},
					}
				},
			},
			{
				text: "$dividend C:D 100.0 CHF",
				want: func(t string) syntax.Booking {
					return syntax.Booking{
						Pos:         syntax.Pos{Start: 0, End: 23, Text: t},
						CreditMacro: syntax.AccountMacro{Start: 0, End: 9, Text: t},
						Debit:       syntax.Account{Start: 10, End: 13, Text: t},
						Amount:      syntax.Decimal{Start: 14, End: 19, Text: t},
						Commodity:   syntax.Commodity{Start: 20, End: 23, Text: t},
					}
				},
			},
			{
				text: "A:B C:D 100.0",
				want: func(t string) syntax.Booking {
					return syntax.Booking{
						Pos:    syntax.Pos{Start: 0, End: 13, Text: t},
						Credit: syntax.Account{Start: 0, End: 3, Text: t},
						Debit:  syntax.Account{Start: 4, End: 7, Text: t},
						Amount: syntax.Decimal{Start: 8, End: 13, Text: t},
					}
				},
				wantErr: true,
			},
			{
				text: "C:D  $dividend  100.0  CHF",
				want: func(t string) syntax.Booking {
					return syntax.Booking{
						Pos:        syntax.Pos{Start: 0, End: 26, Text: t},
						Credit:     syntax.Account{Start: 0, End: 3, Text: t},
						DebitMacro: syntax.AccountMacro{Start: 5, End: 14, Text: t},
						Amount:     syntax.Decimal{Start: 16, End: 21, Text: t},
						Commodity:  syntax.Commodity{Start: 23, End: 26, Text: t},
					}
				},
			},
		},
		desc: "p.parseBooking()",
		fn: func(p *Parser) (syntax.Booking, error) {
			return p.parseBooking()
		},
	}.run(t)
}
