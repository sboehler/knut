package parser

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/sboehler/knut/lib/syntax"
)

type Range = syntax.Range

type testcase[T any] struct {
	text    string
	want    func(string) T
	wantErr bool
	err     func(string) error
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
				t.Errorf("%s returned error %v, want error presence %t", tests.desc, err, test.wantErr)
			}
			if diff := cmp.Diff(test.want(test.text), got); diff != "" {
				t.Errorf("%s returned unexpected diff (-want/+got)\n%s\n", tests.desc, diff)
			}
		})
	}
}

func (tests parserTest[T]) runE(t *testing.T) {
	t.Helper()
	for _, test := range tests.tests {
		t.Run(test.text, func(t *testing.T) {
			parser := New(test.text, "")
			if err := parser.Advance(); err != nil {
				t.Fatalf("s.Advance() = %v, want nil", err)
			}
			var wantErr error
			if test.err != nil {
				wantErr = test.err(test.text)
			}

			got, err := tests.fn(parser)

			if diff := cmp.Diff(wantErr, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("%s returned unexpected diff in err (-want/+got)\n%s\n", tests.desc, diff)
			}
			if diff := cmp.Diff(test.want(test.text), got); diff != "" {
				t.Errorf("%s returned unexpected diff (-want/+got)\n%s\n", tests.desc, diff)
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
					return syntax.Commodity{Range: Range{Start: 0, End: 6, Text: s}}
				},
			},
			{
				text: "",
				want: func(s string) syntax.Commodity {
					return syntax.Commodity{Range: Range{Start: 0, End: 0, Text: s}}
				},
				wantErr: true,
			},
			{
				text: "(foobar)",
				want: func(s string) syntax.Commodity {
					return syntax.Commodity{Range: Range{Start: 0, End: 0, Text: s}}
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
					return syntax.Account{Range: Range{Start: 0, End: 6, Text: s}}
				},
			},
			{
				text: "",
				want: func(s string) syntax.Account {
					return syntax.Account{Range: Range{Start: 0, End: 0, Text: s}}
				},
				err: func(s string) error {
					return syntax.Error{
						Message: "while parsing account",
						Wrapped: syntax.Error{Message: "unexpected end of file, want a letter or a digit"},
					}
				},
			},
			{
				text: "(foobar)",
				want: func(s string) syntax.Account {
					return syntax.Account{Range: Range{Start: 0, End: 0, Text: s}}
				},
				err: func(s string) error {
					return syntax.Error{
						Message: "while parsing account",
						Range:   syntax.Range{Text: "(foobar)"},
						Wrapped: syntax.Error{
							Range:   syntax.Range{Text: "(foobar)"},
							Message: "unexpected character `(`, want a letter or a digit",
						},
					}
				},
			},
			{
				text: "ABC:",
				want: func(s string) syntax.Account {
					return syntax.Account{Range: Range{Start: 0, End: 4, Text: s}}
				},
				err: func(s string) error {
					return syntax.Error{
						Range:   syntax.Range{End: 4, Text: "ABC:"},
						Message: "while parsing account",
						Wrapped: syntax.Error{
							Range:   syntax.Range{Start: 4, End: 4, Text: "ABC:"},
							Message: "unexpected end of file, want a letter or a digit",
						},
					}
				},
			},
			{
				text: "ABC:B",
				want: func(s string) syntax.Account {
					return syntax.Account{Range: Range{Start: 0, End: 5, Text: s}}
				},
			},
			{
				text: "ABC:B:C:D",
				want: func(s string) syntax.Account {
					return syntax.Account{Range: Range{Start: 0, End: 9, Text: s}}
				},
			},
		},
		desc: "p.parseAccount()",
		fn: func(p *Parser) (syntax.Account, error) {
			return p.parseAccount()
		},
	}.runE(t)
}

func TestParseAccountMacro(t *testing.T) {
	parserTest[syntax.AccountMacro]{
		tests: []testcase[syntax.AccountMacro]{
			{
				text: "$foobar",
				want: func(s string) syntax.AccountMacro {
					return syntax.AccountMacro{Range: Range{Start: 0, End: 7, Text: s}}
				},
			},
			{
				text: "$foo1",
				want: func(s string) syntax.AccountMacro {
					return syntax.AccountMacro{Range: Range{Start: 0, End: 4, Text: s}}
				},
			},
			{
				text: "$1foo",
				want: func(s string) syntax.AccountMacro {
					return syntax.AccountMacro{Range: Range{Start: 0, End: 1, Text: s}}
				},
				err: func(s string) error {
					return syntax.Error{
						Message: "while parsing account macro",
						Range:   syntax.Range{End: 1, Text: "$1foo"},
						Wrapped: syntax.Error{
							Message: "unexpected character `1`, want a letter",
							Range:   syntax.Range{Start: 1, End: 1, Text: "$1foo"},
						},
					}
				},
			},
			{
				text: "",
				want: func(s string) syntax.AccountMacro {
					return syntax.AccountMacro{Range: Range{Start: 0, End: 0, Text: s}}
				},
				err: func(s string) error {
					return syntax.Error{
						Message: "while parsing account macro",
						Range:   syntax.Range{Text: s},
						Wrapped: syntax.Error{
							Message: "unexpected end of file, want `$`",
						},
					}
				},
			},
			{
				text: "foobar",
				want: func(s string) syntax.AccountMacro {
					return syntax.AccountMacro{Range: Range{Start: 0, End: 0, Text: s}}
				},
				err: func(s string) error {
					return syntax.Error{
						Message: "while parsing account macro",
						Range:   syntax.Range{Text: "foobar"},
						Wrapped: syntax.Error{
							Range:   syntax.Range{Text: "foobar"},
							Message: "unexpected character `f`, want `$`",
						},
					}
				},
			},
		},
		desc: "p.parseAccountMacro()",
		fn: func(p *Parser) (syntax.AccountMacro, error) {
			return p.parseAccountMacro()
		},
	}.runE(t)
}

func TestParseDecimal(t *testing.T) {
	parserTest[syntax.Decimal]{
		tests: []testcase[syntax.Decimal]{
			{
				text: "10",
				want: func(s string) syntax.Decimal {
					return syntax.Decimal{Range: Range{Start: 0, End: 2, Text: s}}
				},
			},
			{
				text: "-10",
				want: func(s string) syntax.Decimal {
					return syntax.Decimal{Range: Range{Start: 0, End: 3, Text: s}}
				},
			},
			{
				text: "-10.0",
				want: func(s string) syntax.Decimal {
					return syntax.Decimal{Range: Range{Start: 0, End: 5, Text: s}}
				},
			},
			{
				text: "-10.",
				want: func(s string) syntax.Decimal {
					return syntax.Decimal{Range: Range{Start: 0, End: 4, Text: s}}
				},
				err: func(s string) error {
					return syntax.Error{
						Message: "while parsing decimal",
						Range:   syntax.Range{End: 4, Text: s},
						Wrapped: syntax.Error{
							Range:   syntax.Range{Start: 4, End: 4, Text: "-10."},
							Message: "unexpected end of file, want a digit",
						},
					}
				},
			},
			{
				text: "99.",
				want: func(s string) syntax.Decimal {
					return syntax.Decimal{Range: Range{Start: 0, End: 3, Text: s}}
				},
				err: func(s string) error {
					return syntax.Error{
						Message: "while parsing decimal",
						Range:   syntax.Range{End: 3, Text: s},
						Wrapped: syntax.Error{
							Range:   syntax.Range{Start: 3, End: 3, Text: "99."},
							Message: "unexpected end of file, want a digit",
						},
					}
				},
			},
			{
				text: "foo",
				want: func(s string) syntax.Decimal {
					return syntax.Decimal{Range: Range{Start: 0, End: 0, Text: s}}
				},
				err: func(s string) error {
					return syntax.Error{
						Message: "while parsing decimal",
						Range:   syntax.Range{Text: s},
						Wrapped: syntax.Error{
							Range:   syntax.Range{Text: "foo"},
							Message: "unexpected character `f`, want a digit",
						},
					}
				},
			},
		},
		desc: "p.parseDecimal()",
		fn: func(p *Parser) (syntax.Decimal, error) {
			return p.parseDecimal()
		},
	}.runE(t)
}

func TestParseDate(t *testing.T) {
	parserTest[syntax.Date]{
		tests: []testcase[syntax.Date]{
			{
				text: "2023-05-31",
				want: func(s string) syntax.Date {
					return syntax.Date{Range: Range{Start: 0, End: 10, Text: s}}
				},
			},
			{
				text: "202-05-31",
				want: func(s string) syntax.Date {
					return syntax.Date{Range: Range{Start: 0, End: 3, Text: s}}
				},
				err: func(s string) error {
					return syntax.Error{
						Range:   syntax.Range{End: 3, Text: s},
						Message: "while parsing the date",
						Wrapped: syntax.Error{
							Range:   syntax.Range{Start: 3, End: 3, Text: s},
							Message: "unexpected character `-`, want a digit",
						},
					}
				},
			},
			{
				text: "20205-31",
				want: func(s string) syntax.Date {
					return syntax.Date{Range: Range{Start: 0, End: 4, Text: s}}
				},
				err: func(s string) error {
					return syntax.Error{
						Range:   syntax.Range{End: 4, Text: s},
						Message: "while parsing the date",
						Wrapped: syntax.Error{
							Range:   syntax.Range{Start: 4, End: 4, Text: s},
							Message: "unexpected character `5`, want `-`",
						},
					}
				},
			},
		},
		desc: "p.parseDate()",
		fn: func(p *Parser) (syntax.Date, error) {
			return p.parseDate()
		},
	}.runE(t)
}

func TestParseBooking(t *testing.T) {
	parserTest[syntax.Booking]{
		tests: []testcase[syntax.Booking]{
			{
				text: "A:B C:D 100.0 CHF",
				want: func(t string) syntax.Booking {
					return syntax.Booking{
						Range:     Range{Start: 0, End: 17, Text: t},
						Credit:    syntax.Account{Range: Range{Start: 0, End: 3, Text: t}},
						Debit:     syntax.Account{Range: Range{Start: 4, End: 7, Text: t}},
						Amount:    syntax.Decimal{Range: Range{Start: 8, End: 13, Text: t}},
						Commodity: syntax.Commodity{Range: Range{Start: 14, End: 17, Text: t}},
					}
				},
			},
			{
				text: "$dividend C:D 100.0 CHF",
				want: func(t string) syntax.Booking {
					return syntax.Booking{
						Range:       Range{Start: 0, End: 23, Text: t},
						CreditMacro: syntax.AccountMacro{Range: Range{Start: 0, End: 9, Text: t}},
						Debit:       syntax.Account{Range: Range{Start: 10, End: 13, Text: t}},
						Amount:      syntax.Decimal{Range: Range{Start: 14, End: 19, Text: t}},
						Commodity:   syntax.Commodity{Range: Range{Start: 20, End: 23, Text: t}},
					}
				},
			},
			{
				text: "A:B C:D 100.0",
				want: func(t string) syntax.Booking {
					return syntax.Booking{
						Range:  Range{Start: 0, End: 13, Text: t},
						Credit: syntax.Account{Range: Range{Start: 0, End: 3, Text: t}},
						Debit:  syntax.Account{Range: Range{Start: 4, End: 7, Text: t}},
						Amount: syntax.Decimal{Range: Range{Start: 8, End: 13, Text: t}},
					}
				},
				wantErr: true,
			},
			{
				text: "C:D  $dividend  100.0  CHF",
				want: func(t string) syntax.Booking {
					return syntax.Booking{
						Range:      Range{Start: 0, End: 26, Text: t},
						Credit:     syntax.Account{Range: Range{Start: 0, End: 3, Text: t}},
						DebitMacro: syntax.AccountMacro{Range: Range{Start: 5, End: 14, Text: t}},
						Amount:     syntax.Decimal{Range: Range{Start: 16, End: 21, Text: t}},
						Commodity:  syntax.Commodity{Range: Range{Start: 23, End: 26, Text: t}},
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

func TestParseQuotedString(t *testing.T) {
	parserTest[syntax.QuotedString]{
		desc: "p.parseQuotedString()",
		fn:   func(p *Parser) (syntax.QuotedString, error) { return p.parseQuotedString() },
		tests: []testcase[syntax.QuotedString]{
			{
				text: "\"\"",
				want: func(s string) syntax.QuotedString {
					return syntax.QuotedString{Range: Range{Start: 0, End: 2, Text: s}}
				},
			},
			{
				text: "\"foo",
				want: func(s string) syntax.QuotedString {
					return syntax.QuotedString{Range: Range{Start: 0, End: 4, Text: s}}
				},
				err: func(s string) error {
					return syntax.Error{
						Message: "while parsing quoted string",
						Range:   Range{Start: 0, End: 4, Text: s},
						Wrapped: syntax.Error{
							Range:   Range{Start: 4, End: 4, Text: s},
							Message: "unexpected end of file, want `\"`",
						},
					}
				},
			},
			{
				text: "\"foo\"",
				want: func(s string) syntax.QuotedString {
					return syntax.QuotedString{Range: Range{Start: 0, End: 5, Text: s}}
				},
			},
			{
				text: "foo",
				want: func(s string) syntax.QuotedString {
					return syntax.QuotedString{Range: Range{Start: 0, End: 0, Text: s}}
				},
				err: func(s string) error {
					return syntax.Error{
						Message: "while parsing quoted string",
						Range:   Range{Start: 0, End: 0, Text: s},
						Wrapped: syntax.Error{
							Range:   Range{Start: 0, End: 0, Text: s},
							Message: "unexpected character `f`, want `\"`",
						},
					}
				},
			},
		},
	}.runE(t)
}

func TestParseTransaction(t *testing.T) {
	parserTest[syntax.Transaction]{
		tests: []testcase[syntax.Transaction]{
			{
				text: "\"foo\"\n" + "A B 1 CHF\n", // 6 + 10
				want: func(t string) syntax.Transaction {
					return syntax.Transaction{
						Range:       Range{Start: 0, End: 16, Text: t},
						Description: syntax.QuotedString{Range: Range{Start: 0, End: 5, Text: t}},
						Bookings: []syntax.Booking{
							{
								Range:     Range{Start: 6, End: 15, Text: t},
								Credit:    syntax.Account{Range: Range{Start: 6, End: 7, Text: t}},
								Debit:     syntax.Account{Range: Range{Start: 8, End: 9, Text: t}},
								Amount:    syntax.Decimal{Range: Range{Start: 10, End: 11, Text: t}},
								Commodity: syntax.Commodity{Range: Range{Start: 12, End: 15, Text: t}},
							},
						},
					}
				},
			},
			{
				text: "\"foo\"\n" + "A B 1 CHF\n" + "B A 1 CHF\n", // 6 + 10 + 10
				want: func(t string) syntax.Transaction {
					return syntax.Transaction{
						Range:       Range{Start: 0, End: 26, Text: t},
						Description: syntax.QuotedString{Range: Range{Start: 0, End: 5, Text: t}},
						Bookings: []syntax.Booking{
							{
								Range:     Range{Start: 6, End: 15, Text: t},
								Credit:    syntax.Account{Range: Range{Start: 6, End: 7, Text: t}},
								Debit:     syntax.Account{Range: Range{Start: 8, End: 9, Text: t}},
								Amount:    syntax.Decimal{Range: Range{Start: 10, End: 11, Text: t}},
								Commodity: syntax.Commodity{Range: Range{Start: 12, End: 15, Text: t}},
							},
							{
								Range:     Range{Start: 16, End: 25, Text: t},
								Credit:    syntax.Account{Range: Range{Start: 16, End: 17, Text: t}},
								Debit:     syntax.Account{Range: Range{Start: 18, End: 19, Text: t}},
								Amount:    syntax.Decimal{Range: Range{Start: 20, End: 21, Text: t}},
								Commodity: syntax.Commodity{Range: Range{Start: 22, End: 25, Text: t}},
							},
						},
					}
				},
			},
			{
				text:    "\"foo\"\n" + "A B 1 CHF", // 6 + 10
				wantErr: true,
				want: func(t string) syntax.Transaction {
					return syntax.Transaction{
						Range:       Range{Start: 0, End: 15, Text: t},
						Description: syntax.QuotedString{Range: Range{Start: 0, End: 5, Text: t}},
						Bookings: []syntax.Booking{
							{
								Range:     Range{Start: 6, End: 15, Text: t},
								Credit:    syntax.Account{Range: Range{Start: 6, End: 7, Text: t}},
								Debit:     syntax.Account{Range: Range{Start: 8, End: 9, Text: t}},
								Amount:    syntax.Decimal{Range: Range{Start: 10, End: 11, Text: t}},
								Commodity: syntax.Commodity{Range: Range{Start: 12, End: 15, Text: t}},
							},
						},
					}
				},
			},
		},
		desc: "p.parseTransaction()",
		fn: func(p *Parser) (syntax.Transaction, error) {
			return p.parseTransaction(syntax.Date{}, syntax.Addons{})
		},
	}.run(t)
}

func TestParseRestOfWhitespaceLine(t *testing.T) {
	parserTest[Range]{
		desc: "p.parseQuotedString()",
		fn:   func(p *Parser) (Range, error) { return p.readRestOfWhitespaceLine() },
		tests: []testcase[Range]{
			{
				text: "\n",
				want: func(s string) Range {
					return Range{Start: 0, End: 1, Text: s}
				},
			},
			{
				text: " \n",
				want: func(s string) Range {
					return Range{Start: 0, End: 2, Text: s}
				},
			},
			{
				text: " foo",
				want: func(s string) Range {
					return Range{Start: 0, End: 1, Text: s}
				},
				wantErr: true,
			},
		},
	}.run(t)
}

func TestReadWhitespace1(t *testing.T) {
	parserTest[Range]{
		desc: "p.readWhitespace1()",
		fn:   func(p *Parser) (Range, error) { return p.readWhitespace1() },
		tests: []testcase[Range]{
			{
				text: "\n",
				want: func(s string) Range {
					return Range{Start: 0, End: 0, Text: s}
				},
			},
			{
				text: " \n",
				want: func(s string) Range {
					return Range{Start: 0, End: 1, Text: s}
				},
			},
			{
				text: " foo",
				want: func(s string) Range {
					return Range{Start: 0, End: 1, Text: s}
				},
			},
			{
				text: "   foo",
				want: func(s string) Range {
					return Range{Start: 0, End: 3, Text: s}
				},
			},
			{
				text: "foo",
				want: func(s string) Range {
					return Range{Start: 0, End: 0, Text: s}
				},
				err: func(s string) error {
					return syntax.Error{
						Message: "unexpected character `f`, want whitespace or a newline",
						Range:   syntax.Range{Text: s},
					}
				},
			},
		},
	}.runE(t)
}
