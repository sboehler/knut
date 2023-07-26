package parser

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/sboehler/knut/lib/syntax"
)

type Range = syntax.Range

type testcase[T any] struct {
	text string
	want func(string) T
	err  func(string) error
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
				err: func(s string) error {
					return syntax.Error{
						Message: "while parsing commodity",
						Range:   Range{Text: s},
						Wrapped: syntax.Error{
							Message: "unexpected end of file, want a letter or a digit",
							Range:   Range{Text: s},
						},
					}
				},
			},
			{
				text: "(foobar)",
				want: func(s string) syntax.Commodity {
					return syntax.Commodity{Range: Range{Start: 0, End: 0, Text: s}}
				},
				err: func(s string) error {
					return syntax.Error{
						Message: "while parsing commodity",
						Range:   Range{Text: s},
						Wrapped: syntax.Error{
							Message: "unexpected character `(`, want a letter or a digit",
							Range:   Range{Text: s},
						},
					}
				},
			},
		},
		fn: func(p *Parser) (syntax.Commodity, error) {
			return p.parseCommodity()
		},
		desc: "p.parseCommodity()",
	}.run(t)
}

func TestParsePerformance(t *testing.T) {
	parserTest[syntax.Performance]{
		tests: []testcase[syntax.Performance]{
			{
				text: "(USD   ,   CHF,GBP)",
				want: func(s string) syntax.Performance {
					return syntax.Performance{
						Range: Range{Start: 0, End: 19, Text: s},
						Targets: []syntax.Commodity{
							{Range: syntax.Range{Start: 1, End: 4, Text: s}},
							{Range: syntax.Range{Start: 11, End: 14, Text: s}},
							{Range: syntax.Range{Start: 15, End: 18, Text: s}},
						},
					}
				},
			},
			{
				text: "(  )",
				want: func(s string) syntax.Performance {
					return syntax.Performance{
						Range: Range{Start: 0, End: 4, Text: s},
					}
				},
			},
			{
				text: "(A)",
				want: func(s string) syntax.Performance {
					return syntax.Performance{
						Range: Range{Start: 0, End: 3, Text: s},
						Targets: []syntax.Commodity{
							{Range: syntax.Range{Start: 1, End: 2, Text: s}},
						},
					}
				},
			},
			{
				text: "",
				want: func(s string) syntax.Performance {
					return syntax.Performance{Range: Range{Start: 0, End: 0, Text: s}}
				},
				err: func(s string) error {
					return syntax.Error{
						Message: "while parsing performance",
						Range:   Range{Text: s},
						Wrapped: syntax.Error{
							Message: "unexpected end of file, want `(`",
							Range:   Range{Text: s},
						},
					}
				},
			},
			{
				text: "(foobar)",
				want: func(s string) syntax.Performance {
					return syntax.Performance{
						Range: Range{Start: 0, End: 8, Text: s},
						Targets: []syntax.Commodity{
							{Range: Range{Start: 1, End: 7, Text: s}},
						},
					}
				},
			},
		},
		fn: func(p *Parser) (syntax.Performance, error) {
			return p.parsePerformance()
		},
		desc: "p.parsePerformance()",
	}.run(t)
}

func TestParseAccrual(t *testing.T) {
	parserTest[syntax.Accrual]{
		tests: []testcase[syntax.Accrual]{
			{
				text: " monthly 2023-01-01 2023-12-31 A:B",
				want: func(s string) syntax.Accrual {
					return syntax.Accrual{
						Range:    syntax.Range{Start: 0, End: 34, Text: s},
						Interval: syntax.Interval{Range: syntax.Range{Start: 1, End: 8, Text: s}},
						Start:    syntax.Date{Range: syntax.Range{Start: 9, End: 19, Text: s}},
						End:      syntax.Date{Range: syntax.Range{Start: 20, End: 30, Text: s}},
						Account:  syntax.Account{Range: syntax.Range{Start: 31, End: 34, Text: s}},
					}
				},
			},
			{
				text: "",
				want: func(s string) syntax.Accrual {
					return syntax.Accrual{
						Range: syntax.Range{Start: 0, End: 0, Text: s}}
				},
				err: func(s string) error {
					return syntax.Error{
						Message: "while parsing addons",
						Range:   syntax.Range{Text: s},
						Wrapped: syntax.Error{
							Message: "while parsing interval",
							Wrapped: syntax.Error{
								Message: "unexpected end of file, want one of {`daily`, `weekly`, `monthly`, `quarterly`}",
							},
						},
					}
				},
			},
		},
		fn: func(p *Parser) (syntax.Accrual, error) {
			return p.parseAccrual()
		},
		desc: "p.parseAccrual()",
	}.run(t)
}

func TestParseAddons(t *testing.T) {
	parserTest[syntax.Addons]{
		tests: []testcase[syntax.Addons]{
			{
				text: "@accrue monthly 2023-01-01  2023-12-31 A:B",
				want: func(s string) syntax.Addons {
					return syntax.Addons{
						Range: syntax.Range{Start: 0, End: 42, Text: s},
						Accrual: syntax.Accrual{
							Range:    syntax.Range{Start: 0, End: 42, Text: s},
							Interval: syntax.Interval{Range: syntax.Range{Start: 8, End: 15, Text: s}},
							Start:    syntax.Date{Range: syntax.Range{Start: 16, End: 26, Text: s}},
							End:      syntax.Date{Range: syntax.Range{Start: 28, End: 38, Text: s}},
							Account:  syntax.Account{Range: syntax.Range{Start: 39, End: 42, Text: s}},
						},
					}
				},
			},
			{
				text: "@performance(USD)",
				want: func(s string) syntax.Addons {
					return syntax.Addons{
						Range: syntax.Range{Start: 0, End: 17, Text: s},
						Performance: syntax.Performance{
							Range: syntax.Range{Start: 0, End: 17, Text: s},
							Targets: []syntax.Commodity{
								{Range: Range{Start: 13, End: 16, Text: s}},
							},
						},
					}
				},
			},
			{
				text: "@performance(USD)\n@accrue daily 2023-01-01 2023-01-01 B:A",
				want: func(s string) syntax.Addons {
					return syntax.Addons{
						Range: syntax.Range{Start: 0, End: 57, Text: s},
						Performance: syntax.Performance{
							Range: syntax.Range{Start: 0, End: 17, Text: s},
							Targets: []syntax.Commodity{
								{Range: Range{Start: 13, End: 16, Text: s}},
							},
						},
						Accrual: syntax.Accrual{
							Range:    syntax.Range{Start: 18, End: 57, Text: s},
							Interval: syntax.Interval{Range: syntax.Range{Start: 26, End: 31, Text: s}},
							Start:    syntax.Date{Range: syntax.Range{Start: 32, End: 42, Text: s}},
							End:      syntax.Date{Range: syntax.Range{Start: 43, End: 53, Text: s}},
							Account:  syntax.Account{Range: syntax.Range{Start: 54, End: 57, Text: s}},
						},
					}
				},
			},
			{
				text: "@performance(USD)\n@performance(CHF)",
				want: func(s string) syntax.Addons {
					return syntax.Addons{
						Range: syntax.Range{Start: 0, End: 30, Text: s},
						Performance: syntax.Performance{
							Range: syntax.Range{Start: 0, End: 17, Text: s},
							Targets: []syntax.Commodity{
								{Range: Range{Start: 13, End: 16, Text: s}},
							},
						},
					}
				},
				err: func(s string) error {
					return syntax.Error{
						Message: "while parsing addons",
						Range:   syntax.Range{End: 30, Text: s},
						Wrapped: syntax.Error{
							Range:   syntax.Range{Start: 18, End: 30, Text: s},
							Message: "duplicate @performance annotation",
						},
					}
				},
			},
			{
				text: "",
				want: func(s string) syntax.Addons {
					return syntax.Addons{
						Range: syntax.Range{Start: 0, End: 0, Text: s}}
				},
				err: func(s string) error {
					return syntax.Error{
						Message: "while parsing addons",
						Range:   syntax.Range{Text: s},
						Wrapped: syntax.Error{
							Message: "unexpected end of file, want one of {`@performance`, `@accrue`}",
						},
					}
				},
			},
		},
		fn: func(p *Parser) (syntax.Addons, error) {
			return p.parseAddons()
		},
		desc: "p.parseAddons()",
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
						Range:   syntax.Range{Text: s},
						Wrapped: syntax.Error{
							Range:   syntax.Range{Text: s},
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
						Range:   syntax.Range{End: 4, Text: s},
						Message: "while parsing account",
						Wrapped: syntax.Error{
							Range:   syntax.Range{Start: 4, End: 4, Text: s},
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
	}.run(t)
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
						Range:   syntax.Range{End: 1, Text: s},
						Wrapped: syntax.Error{
							Message: "unexpected character `1`, want a letter",
							Range:   syntax.Range{Start: 1, End: 1, Text: s},
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
						Range:   syntax.Range{Text: s},
						Wrapped: syntax.Error{
							Range:   syntax.Range{Text: s},
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
	}.run(t)
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
							Range:   syntax.Range{Start: 4, End: 4, Text: s},
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
							Range:   syntax.Range{Start: 3, End: 3, Text: s},
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
							Range:   syntax.Range{Text: s},
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
	}.run(t)
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
	}.run(t)
}

func TestParseInterval(t *testing.T) {
	parserTest[syntax.Interval]{
		tests: []testcase[syntax.Interval]{
			{
				text: "daily",
				want: func(s string) syntax.Interval {
					return syntax.Interval{Range: Range{Start: 0, End: 5, Text: s}}
				},
			},
			{
				text: "weekly",
				want: func(s string) syntax.Interval {
					return syntax.Interval{Range: Range{Start: 0, End: 6, Text: s}}
				},
			},
			{
				text: "monthly",
				want: func(s string) syntax.Interval {
					return syntax.Interval{Range: Range{Start: 0, End: 7, Text: s}}
				},
			},
			{
				text: "quarterly",
				want: func(s string) syntax.Interval {
					return syntax.Interval{Range: Range{Start: 0, End: 9, Text: s}}
				},
			},
			{
				text: "",
				want: func(s string) syntax.Interval {
					return syntax.Interval{Range: Range{Start: 0, End: 0, Text: s}}
				},
				err: func(s string) error {
					return syntax.Error{
						Range:   syntax.Range{Text: s},
						Message: "while parsing interval",
						Wrapped: syntax.Error{
							Range:   syntax.Range{Text: s},
							Message: "unexpected end of file, want one of {`daily`, `weekly`, `monthly`, `quarterly`}",
						},
					}
				},
			},
		},
		desc: "p.parseInterval()",
		fn: func(p *Parser) (syntax.Interval, error) {
			return p.parseInterval()
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
				err: func(s string) error {
					return syntax.Error{
						Message: "while parsing booking",
						Range:   Range{Start: 0, End: 13, Text: s},
						Wrapped: syntax.Error{
							Range:   syntax.Range{Start: 13, End: 13, Text: s},
							Message: "unexpected end of file, want whitespace",
						}}
				},
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
	}.run(t)
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
				text: "\"foo\"\n" + "A B 1 CHF", // 6 + 10
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
			{
				text: strings.Join([]string{`"foo"`, "A B"}, "\n"), // 6 + 10
				want: func(t string) syntax.Transaction {
					return syntax.Transaction{
						Range:       Range{Start: 0, End: 9, Text: t},
						Description: syntax.QuotedString{Range: Range{Start: 0, End: 5, Text: t}},
						Bookings: []syntax.Booking{
							{
								Range:  Range{Start: 6, End: 9, Text: t},
								Credit: syntax.Account{Range: Range{Start: 6, End: 7, Text: t}},
								Debit:  syntax.Account{Range: Range{Start: 8, End: 9, Text: t}},
							},
						},
					}
				},
				err: func(s string) error {
					return syntax.Error{
						Message: "while parsing transaction",
						Range:   Range{Start: 0, End: 9, Text: s},
						Wrapped: syntax.Error{
							Range:   syntax.Range{Start: 6, End: 9, Text: s},
							Message: "while parsing booking",
							Wrapped: syntax.Error{
								Range:   syntax.Range{Start: 9, End: 9, Text: s},
								Message: "unexpected end of file, want whitespace",
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
				err: func(s string) error {
					return syntax.Error{
						Message: "unexpected character `f`, want `\n`",
						Range:   syntax.Range{Start: 1, End: 1, Text: s},
					}
				},
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
	}.run(t)
}
