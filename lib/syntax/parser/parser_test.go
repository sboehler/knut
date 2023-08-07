package parser

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/sboehler/knut/lib/syntax/directives"
)

type Range = directives.Range

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

			if diff := cmp.Diff(test.want(test.text), got); diff != "" {
				t.Errorf("%s returned unexpected diff (-want/+got)\n%s\n", tests.desc, diff)
			}
			if diff := cmp.Diff(wantErr, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("%s returned unexpected diff in err (-want/+got)\n%s\n", tests.desc, diff)
			}
		})
	}
}

func TestParseFile(t *testing.T) {
	parserTest[directives.File]{
		tests: []testcase[directives.File]{
			{
				text: strings.Join([]string{
					"",
					"",
					"include \"foo.knut\"",
					"2021-01-01 open A",
					"# comment",
				}, "\n"),
				want: func(s string) directives.File {
					return directives.File{
						Range: Range{End: 48, Text: s},
						Directives: []directives.Directive{
							{
								Range: Range{Start: 2, End: 20, Text: s},
								Directive: directives.Include{
									Range: directives.Range{Start: 2, End: 20, Text: s},
									IncludePath: directives.QuotedString{
										Range:   Range{Start: 10, End: 20, Text: s},
										Content: Range{Start: 11, End: 19, Text: s},
									},
								},
							},
							{
								Range: Range{Start: 21, End: 38, Text: s},
								Directive: directives.Open{
									Range: directives.Range{Start: 21, End: 38, Text: s},
									Date: directives.Date{
										Range: Range{Start: 21, End: 31, Text: s},
									},
									Account: directives.Account{
										Range: Range{Start: 37, End: 38, Text: s},
									},
								},
							},
						},
					}
				},
			},
			{
				text: strings.Join([]string{
					"",
					"asdf",
				}, "\n"),
				want: func(s string) directives.File {
					return directives.File{
						Range: Range{End: 1, Text: s},
						Directives: []directives.Directive{
							{Range: directives.Range{Start: 1, End: 1, Text: s}},
						},
					}
				},
				err: func(s string) error {
					return directives.Error{
						Message: "while parsing file ``",
						Range:   Range{End: 1, Text: s},
						Wrapped: directives.Error{
							Message: "while parsing directive",
							Range:   Range{Start: 1, End: 1, Text: s},
							Wrapped: directives.Error{
								Message: "while parsing the date",
								Range:   Range{Start: 1, End: 1, Text: s},
								Wrapped: directives.Error{
									Range:   directives.Range{Start: 1, End: 1, Text: s},
									Message: "unexpected character `a`, want a digit",
								},
							},
						},
					}
				},
			},
			{
				text: strings.Join([]string{
					"", "", "", "",
					"  include \"foo\"",
				}, "\n"),
				want: func(s string) directives.File {
					return directives.File{
						Range: Range{End: 6, Text: s},
					}
				},
				err: func(s string) error {
					return directives.Error{
						Message: "while parsing file ``",
						Range:   Range{End: 6, Text: s},
						Wrapped: directives.Error{
							Range:   directives.Range{Start: 4, End: 6, Text: "\n\n\n\n  include \"foo\""},
							Message: "while reading the rest of the line",
							Wrapped: directives.Error{
								Range:   directives.Range{Start: 6, End: 6, Text: s},
								Message: "unexpected character `i`, want `\n`",
							},
						},
					}
				},
			},
			{
				text: strings.Join([]string{
					`2022-03-03 "Hello, world"`,
					`A:B:C C:B:ASDF 400 CHF`,
				}, "\n"),
				want: func(s string) directives.File {
					return directives.File{
						Range: Range{End: 48, Text: s},
						Directives: []directives.Directive{
							{
								Range: directives.Range{End: 48, Text: s},
								Directive: directives.Transaction{
									Range: directives.Range{End: 48, Text: s},
									Date:  directives.Date{Range: Range{End: 10, Text: s}},
									Description: directives.QuotedString{
										Range:   Range{Start: 11, End: 25, Text: s},
										Content: Range{Start: 12, End: 24, Text: s},
									},
									Bookings: []directives.Booking{
										{
											Range: directives.Range{Start: 26, End: 48, Text: s},
											Credit: directives.Account{
												Range: directives.Range{Start: 26, End: 31, Text: s},
											},
											Debit: directives.Account{
												Range: directives.Range{Start: 32, End: 40, Text: s},
											},
											Amount: directives.Decimal{
												Range: directives.Range{Start: 41, End: 44, Text: s},
											},
											Commodity: directives.Commodity{
												Range: directives.Range{Start: 45, End: 48, Text: s},
											},
										},
									},
								},
							},
						},
					}
				},
			},
		},
		desc: "p.parseFile()",
		fn: func(p *Parser) (directives.File, error) {
			return p.ParseFile()
		},
	}.run(t)
}

func TestParseCommodity(t *testing.T) {
	parserTest[directives.Commodity]{
		tests: []testcase[directives.Commodity]{
			{
				text: "foobar",
				want: func(s string) directives.Commodity {
					return directives.Commodity{Range: Range{End: 6, Text: s}}
				},
			},
			{
				text: "",
				want: func(s string) directives.Commodity {
					return directives.Commodity{Range: Range{Text: s}}
				},
				err: func(s string) error {
					return directives.Error{
						Message: "while parsing commodity",
						Range:   Range{Text: s},
						Wrapped: directives.Error{
							Message: "unexpected end of file, want a letter or a digit",
							Range:   Range{Text: s},
						},
					}
				},
			},
			{
				text: "(foobar)",
				want: func(s string) directives.Commodity {
					return directives.Commodity{Range: Range{Text: s}}
				},
				err: func(s string) error {
					return directives.Error{
						Message: "while parsing commodity",
						Range:   Range{Text: s},
						Wrapped: directives.Error{
							Message: "unexpected character `(`, want a letter or a digit",
							Range:   Range{Text: s},
						},
					}
				},
			},
		},
		fn: func(p *Parser) (directives.Commodity, error) {
			return p.parseCommodity()
		},
		desc: "p.parseCommodity()",
	}.run(t)
}

func TestParsePerformance(t *testing.T) {
	parserTest[directives.Performance]{
		tests: []testcase[directives.Performance]{
			{
				text: "(USD   ,   CHF,GBP)",
				want: func(s string) directives.Performance {
					return directives.Performance{
						Range: Range{End: 19, Text: s},
						Targets: []directives.Commodity{
							{Range: Range{Start: 1, End: 4, Text: s}},
							{Range: Range{Start: 11, End: 14, Text: s}},
							{Range: Range{Start: 15, End: 18, Text: s}},
						},
					}
				},
			},
			{
				text: "(  )",
				want: func(s string) directives.Performance {
					return directives.Performance{
						Range: Range{End: 4, Text: s},
					}
				},
			},
			{
				text: "(A)",
				want: func(s string) directives.Performance {
					return directives.Performance{
						Range: Range{End: 3, Text: s},
						Targets: []directives.Commodity{
							{Range: Range{Start: 1, End: 2, Text: s}},
						},
					}
				},
			},
			{
				text: "",
				want: func(s string) directives.Performance {
					return directives.Performance{Range: Range{Text: s}}
				},
				err: func(s string) error {
					return directives.Error{
						Message: "while parsing performance",
						Range:   Range{Text: s},
						Wrapped: directives.Error{
							Message: "unexpected end of file, want `(`",
							Range:   Range{Text: s},
						},
					}
				},
			},
			{
				text: "(foobar)",
				want: func(s string) directives.Performance {
					return directives.Performance{
						Range: Range{End: 8, Text: s},
						Targets: []directives.Commodity{
							{Range: Range{Start: 1, End: 7, Text: s}},
						},
					}
				},
			},
		},
		fn: func(p *Parser) (directives.Performance, error) {
			return p.parsePerformance()
		},
		desc: "p.parsePerformance()",
	}.run(t)
}

func TestParseAccrual(t *testing.T) {
	parserTest[directives.Accrual]{
		tests: []testcase[directives.Accrual]{
			{
				text: " monthly 2023-01-01 2023-12-31 A:B",
				want: func(s string) directives.Accrual {
					return directives.Accrual{
						Range:    Range{End: 34, Text: s},
						Interval: directives.Interval{Range: Range{Start: 1, End: 8, Text: s}},
						Start:    directives.Date{Range: Range{Start: 9, End: 19, Text: s}},
						End:      directives.Date{Range: Range{Start: 20, End: 30, Text: s}},
						Account:  directives.Account{Range: Range{Start: 31, End: 34, Text: s}},
					}
				},
			},
			{
				text: "",
				want: func(s string) directives.Accrual {
					return directives.Accrual{
						Range: Range{Text: s}}
				},
				err: func(s string) error {
					return directives.Error{
						Message: "while parsing addons",
						Range:   Range{Text: s},
						Wrapped: directives.Error{
							Message: "while parsing interval",
							Wrapped: directives.Error{
								Message: "unexpected end of file, want one of {`daily`, `weekly`, `monthly`, `quarterly`}",
							},
						},
					}
				},
			},
		},
		fn: func(p *Parser) (directives.Accrual, error) {
			return p.parseAccrual()
		},
		desc: "p.parseAccrual()",
	}.run(t)
}

func TestParseAddons(t *testing.T) {
	parserTest[directives.Addons]{
		tests: []testcase[directives.Addons]{
			{
				text: "@accrue monthly 2023-01-01  2023-12-31 A:B",
				want: func(s string) directives.Addons {
					return directives.Addons{
						Range: Range{End: 42, Text: s},
						Accrual: directives.Accrual{
							Range:    Range{End: 42, Text: s},
							Interval: directives.Interval{Range: Range{Start: 8, End: 15, Text: s}},
							Start:    directives.Date{Range: Range{Start: 16, End: 26, Text: s}},
							End:      directives.Date{Range: Range{Start: 28, End: 38, Text: s}},
							Account:  directives.Account{Range: Range{Start: 39, End: 42, Text: s}},
						},
					}
				},
			},
			{
				text: "@performance(USD)",
				want: func(s string) directives.Addons {
					return directives.Addons{
						Range: Range{End: 17, Text: s},
						Performance: directives.Performance{
							Range: Range{End: 17, Text: s},
							Targets: []directives.Commodity{
								{Range: Range{Start: 13, End: 16, Text: s}},
							},
						},
					}
				},
			},
			{
				text: "@performance(USD)\n@accrue daily 2023-01-01 2023-01-01 B:A",
				want: func(s string) directives.Addons {
					return directives.Addons{
						Range: Range{End: 57, Text: s},
						Performance: directives.Performance{
							Range: Range{End: 17, Text: s},
							Targets: []directives.Commodity{
								{Range: Range{Start: 13, End: 16, Text: s}},
							},
						},
						Accrual: directives.Accrual{
							Range:    Range{Start: 18, End: 57, Text: s},
							Interval: directives.Interval{Range: Range{Start: 26, End: 31, Text: s}},
							Start:    directives.Date{Range: Range{Start: 32, End: 42, Text: s}},
							End:      directives.Date{Range: directives.Range{Start: 43, End: 53, Text: s}},
							Account:  directives.Account{Range: directives.Range{Start: 54, End: 57, Text: s}},
						},
					}
				},
			},
			{
				text: "@performance(USD)\n@performance(CHF)",
				want: func(s string) directives.Addons {
					return directives.Addons{
						Range: directives.Range{End: 30, Text: s},
						Performance: directives.Performance{
							Range: directives.Range{End: 17, Text: s},
							Targets: []directives.Commodity{
								{Range: Range{Start: 13, End: 16, Text: s}},
							},
						},
					}
				},
				err: func(s string) error {
					return directives.Error{
						Message: "while parsing addons",
						Range:   directives.Range{End: 30, Text: s},
						Wrapped: directives.Error{
							Range:   directives.Range{Start: 18, End: 30, Text: s},
							Message: "duplicate performance annotation",
						},
					}
				},
			},
			{
				text: "@accrue daily 2023-01-01 2023-12-31 B\n@accrue daily 2023-01-01 2023-12-31 B",
				want: func(s string) directives.Addons {
					return directives.Addons{
						Range: directives.Range{End: 45, Text: s},
						Accrual: directives.Accrual{
							Range:    directives.Range{End: 37, Text: s},
							Interval: directives.Interval{Range: Range{Start: 8, End: 13, Text: s}},
							Start:    directives.Date{Range: Range{Start: 14, End: 24, Text: s}},
							End:      directives.Date{Range: Range{Start: 25, End: 35, Text: s}},
							Account:  directives.Account{Range: Range{Start: 36, End: 37, Text: s}},
						},
					}
				},
				err: func(s string) error {
					return directives.Error{
						Range:   directives.Range{End: 45, Text: s},
						Message: "while parsing addons",
						Wrapped: directives.Error{
							Range:   directives.Range{Start: 38, End: 45, Text: s},
							Message: "duplicate accrue annotation",
						},
					}
				},
			},
			{
				text: "",
				want: func(s string) directives.Addons {
					return directives.Addons{
						Range: directives.Range{Text: s}}
				},
				err: func(s string) error {
					return directives.Error{
						Message: "while parsing addons",
						Range:   directives.Range{Text: s},
						Wrapped: directives.Error{
							Message: "unexpected end of file, want one of {`@performance`, `@accrue`}",
						},
					}
				},
			},
		},
		fn: func(p *Parser) (directives.Addons, error) {
			return p.parseAddons()
		},
		desc: "p.parseAddons()",
	}.run(t)
}

func TestParseAccount(t *testing.T) {
	parserTest[directives.Account]{
		tests: []testcase[directives.Account]{
			{
				text: "foobar",
				want: func(s string) directives.Account {
					return directives.Account{Range: Range{End: 6, Text: s}}
				},
			},
			{
				text: "",
				want: func(s string) directives.Account {
					return directives.Account{Range: Range{Text: s}}
				},
				err: func(s string) error {
					return directives.Error{
						Message: "while parsing account",
						Wrapped: directives.Error{Message: "unexpected end of file, want a letter or a digit"},
					}
				},
			},
			{
				text: "(foobar)",
				want: func(s string) directives.Account {
					return directives.Account{Range: Range{Text: s}}
				},
				err: func(s string) error {
					return directives.Error{
						Message: "while parsing account",
						Range:   directives.Range{Text: s},
						Wrapped: directives.Error{
							Range:   directives.Range{Text: s},
							Message: "unexpected character `(`, want a letter or a digit",
						},
					}
				},
			},
			{
				text: "ABC:",
				want: func(s string) directives.Account {
					return directives.Account{Range: Range{End: 4, Text: s}}
				},
				err: func(s string) error {
					return directives.Error{
						Range:   directives.Range{End: 4, Text: s},
						Message: "while parsing account",
						Wrapped: directives.Error{
							Range:   directives.Range{Start: 4, End: 4, Text: s},
							Message: "unexpected end of file, want a letter or a digit",
						},
					}
				},
			},
			{
				text: "ABC:B",
				want: func(s string) directives.Account {
					return directives.Account{Range: Range{End: 5, Text: s}}
				},
			},
			{
				text: "ABC:B:C:D",
				want: func(s string) directives.Account {
					return directives.Account{Range: Range{End: 9, Text: s}}
				},
			},
			{
				text: "$foobar",
				want: func(s string) directives.Account {
					return directives.Account{
						Range: Range{End: 7, Text: s},
						Macro: true,
					}
				},
			},
			{
				text: "$foo1",
				want: func(s string) directives.Account {
					return directives.Account{
						Range: Range{End: 4, Text: s},
						Macro: true,
					}
				},
			},
			{
				text: "$1foo",
				want: func(s string) directives.Account {
					return directives.Account{
						Range: Range{End: 1, Text: s},
						Macro: true,
					}
				},
				err: func(s string) error {
					return directives.Error{
						Message: "while parsing account",
						Range:   directives.Range{End: 1, Text: s},
						Wrapped: directives.Error{
							Message: "unexpected character `1`, want a letter",
							Range:   directives.Range{Start: 1, End: 1, Text: s},
						},
					}
				},
			},
		},
		desc: "p.parseAccount()",
		fn: func(p *Parser) (directives.Account, error) {
			return p.parseAccount()
		},
	}.run(t)
}

func TestParseDecimal(t *testing.T) {
	parserTest[directives.Decimal]{
		tests: []testcase[directives.Decimal]{
			{
				text: "10",
				want: func(s string) directives.Decimal {
					return directives.Decimal{Range: Range{End: 2, Text: s}}
				},
			},
			{
				text: "-10",
				want: func(s string) directives.Decimal {
					return directives.Decimal{Range: Range{End: 3, Text: s}}
				},
			},
			{
				text: "-10.0",
				want: func(s string) directives.Decimal {
					return directives.Decimal{Range: Range{End: 5, Text: s}}
				},
			},
			{
				text: "-10.",
				want: func(s string) directives.Decimal {
					return directives.Decimal{Range: Range{End: 4, Text: s}}
				},
				err: func(s string) error {
					return directives.Error{
						Message: "while parsing decimal",
						Range:   directives.Range{End: 4, Text: s},
						Wrapped: directives.Error{
							Range:   directives.Range{Start: 4, End: 4, Text: s},
							Message: "unexpected end of file, want a digit",
						},
					}
				},
			},
			{
				text: "99.",
				want: func(s string) directives.Decimal {
					return directives.Decimal{Range: Range{End: 3, Text: s}}
				},
				err: func(s string) error {
					return directives.Error{
						Message: "while parsing decimal",
						Range:   directives.Range{End: 3, Text: s},
						Wrapped: directives.Error{
							Range:   directives.Range{Start: 3, End: 3, Text: s},
							Message: "unexpected end of file, want a digit",
						},
					}
				},
			},
			{
				text: "foo",
				want: func(s string) directives.Decimal {
					return directives.Decimal{Range: Range{Text: s}}
				},
				err: func(s string) error {
					return directives.Error{
						Message: "while parsing decimal",
						Range:   directives.Range{Text: s},
						Wrapped: directives.Error{
							Range:   directives.Range{Text: s},
							Message: "unexpected character `f`, want a digit",
						},
					}
				},
			},
		},
		desc: "p.parseDecimal()",
		fn: func(p *Parser) (directives.Decimal, error) {
			return p.parseDecimal()
		},
	}.run(t)
}

func TestParseDate(t *testing.T) {
	parserTest[directives.Date]{
		tests: []testcase[directives.Date]{
			{
				text: "2023-05-31",
				want: func(s string) directives.Date {
					return directives.Date{Range: Range{End: 10, Text: s}}
				},
			},
			{
				text: "202-05-31",
				want: func(s string) directives.Date {
					return directives.Date{Range: Range{End: 3, Text: s}}
				},
				err: func(s string) error {
					return directives.Error{
						Range:   directives.Range{End: 3, Text: s},
						Message: "while parsing the date",
						Wrapped: directives.Error{
							Range:   directives.Range{Start: 3, End: 3, Text: s},
							Message: "unexpected character `-`, want a digit",
						},
					}
				},
			},
			{
				text: "20205-31",
				want: func(s string) directives.Date {
					return directives.Date{Range: Range{End: 4, Text: s}}
				},
				err: func(s string) error {
					return directives.Error{
						Range:   directives.Range{End: 4, Text: s},
						Message: "while parsing the date",
						Wrapped: directives.Error{
							Range:   directives.Range{Start: 4, End: 4, Text: s},
							Message: "unexpected character `5`, want `-`",
						},
					}
				},
			},
		},
		desc: "p.parseDate()",
		fn: func(p *Parser) (directives.Date, error) {
			return p.parseDate()
		},
	}.run(t)
}

func TestReadComment(t *testing.T) {
	parserTest[directives.Range]{
		tests: []testcase[directives.Range]{
			{
				text: "//foobar\n",
				want: func(s string) directives.Range {
					return directives.Range{End: 8, Text: s}
				},
			},
			{
				text: "#foobar\n",
				want: func(s string) directives.Range {
					return directives.Range{End: 7, Text: s}
				},
			},
			{
				text: "* a comment",
				want: func(s string) directives.Range {
					return directives.Range{End: 11, Text: s}
				},
			},
			{
				text: "-- not a comment",
				want: func(s string) directives.Range {
					return directives.Range{Text: s}
				},
				err: func(s string) error {
					return directives.Error{
						Message: "while reading comment",
						Range:   directives.Range{Text: s},
						Wrapped: directives.Error{
							Message: "unexpected input, want one of {`*`, `//`, `#`}",
							Range:   directives.Range{Text: s},
						},
					}
				},
			},
		},
		desc: "p.readComment()",
		fn: func(p *Parser) (directives.Range, error) {
			return p.readComment()
		},
	}.run(t)
}

func TestParseInterval(t *testing.T) {
	parserTest[directives.Interval]{
		tests: []testcase[directives.Interval]{
			{
				text: "daily",
				want: func(s string) directives.Interval {
					return directives.Interval{Range: Range{End: 5, Text: s}}
				},
			},
			{
				text: "weekly",
				want: func(s string) directives.Interval {
					return directives.Interval{Range: Range{End: 6, Text: s}}
				},
			},
			{
				text: "monthly",
				want: func(s string) directives.Interval {
					return directives.Interval{Range: Range{End: 7, Text: s}}
				},
			},
			{
				text: "quarterly",
				want: func(s string) directives.Interval {
					return directives.Interval{Range: Range{End: 9, Text: s}}
				},
			},
			{
				text: "",
				want: func(s string) directives.Interval {
					return directives.Interval{Range: Range{Text: s}}
				},
				err: func(s string) error {
					return directives.Error{
						Range:   directives.Range{Text: s},
						Message: "while parsing interval",
						Wrapped: directives.Error{
							Range:   directives.Range{Text: s},
							Message: "unexpected end of file, want one of {`daily`, `weekly`, `monthly`, `quarterly`}",
						},
					}
				},
			},
		},
		desc: "p.parseInterval()",
		fn: func(p *Parser) (directives.Interval, error) {
			return p.parseInterval()
		},
	}.run(t)
}

func TestParseBooking(t *testing.T) {
	parserTest[directives.Booking]{
		tests: []testcase[directives.Booking]{
			{
				text: "A:B C:D 100.0 CHF",
				want: func(t string) directives.Booking {
					return directives.Booking{
						Range:     Range{End: 17, Text: t},
						Credit:    directives.Account{Range: Range{End: 3, Text: t}},
						Debit:     directives.Account{Range: Range{Start: 4, End: 7, Text: t}},
						Amount:    directives.Decimal{Range: Range{Start: 8, End: 13, Text: t}},
						Commodity: directives.Commodity{Range: Range{Start: 14, End: 17, Text: t}},
					}
				},
			},
			{
				text: "$dividend C:D 100.0 CHF",
				want: func(t string) directives.Booking {
					return directives.Booking{
						Range:     Range{End: 23, Text: t},
						Credit:    directives.Account{Range: Range{End: 9, Text: t}, Macro: true},
						Debit:     directives.Account{Range: Range{Start: 10, End: 13, Text: t}},
						Amount:    directives.Decimal{Range: Range{Start: 14, End: 19, Text: t}},
						Commodity: directives.Commodity{Range: Range{Start: 20, End: 23, Text: t}},
					}
				},
			},
			{
				text: "A:B C:D 100.0",
				want: func(t string) directives.Booking {
					return directives.Booking{
						Range:  Range{End: 13, Text: t},
						Credit: directives.Account{Range: Range{End: 3, Text: t}},
						Debit:  directives.Account{Range: Range{Start: 4, End: 7, Text: t}},
						Amount: directives.Decimal{Range: Range{Start: 8, End: 13, Text: t}},
					}
				},
				err: func(s string) error {
					return directives.Error{
						Message: "while parsing booking",
						Range:   Range{End: 13, Text: s},
						Wrapped: directives.Error{
							Range:   directives.Range{Start: 13, End: 13, Text: s},
							Message: "unexpected end of file, want whitespace",
						}}
				},
			},
			{
				text: "$$ C:D 100.0",
				want: func(t string) directives.Booking {
					return directives.Booking{
						Range: Range{End: 1, Text: t},
						Credit: directives.Account{
							Range: directives.Range{End: 1, Text: t},
							Macro: true,
						},
					}
				},
				err: func(s string) error {
					return directives.Error{
						Message: "while parsing booking",
						Range:   Range{End: 1, Text: s},
						Wrapped: directives.Error{
							Range:   directives.Range{End: 1, Text: s},
							Message: "while parsing account",
							Wrapped: directives.Error{
								Range:   directives.Range{Start: 1, End: 1, Text: s},
								Message: "unexpected character `$`, want a letter",
							},
						}}
				},
			},
			{
				text: "C:D  $dividend  100.0  CHF",
				want: func(t string) directives.Booking {
					return directives.Booking{
						Range:     Range{End: 26, Text: t},
						Credit:    directives.Account{Range: Range{End: 3, Text: t}},
						Debit:     directives.Account{Range: Range{Start: 5, End: 14, Text: t}, Macro: true},
						Amount:    directives.Decimal{Range: Range{Start: 16, End: 21, Text: t}},
						Commodity: directives.Commodity{Range: Range{Start: 23, End: 26, Text: t}},
					}
				},
			},
		},
		desc: "p.parseBooking()",
		fn: func(p *Parser) (directives.Booking, error) {
			return p.parseBooking()
		},
	}.run(t)
}

func TestParseInclude(t *testing.T) {
	parserTest[directives.Include]{
		tests: []testcase[directives.Include]{
			{
				text: `include "/foo/bar.knut"`,
				want: func(t string) directives.Include {
					return directives.Include{
						Range: Range{End: 23, Text: t},
						IncludePath: directives.QuotedString{
							Range:   Range{Start: 8, End: 23, Text: t},
							Content: Range{Start: 9, End: 22, Text: t},
						},
					}
				},
			},
			{
				text: `incline "foo"`,
				want: func(s string) directives.Include {
					return directives.Include{
						Range: Range{End: 4, Text: s},
					}
				},
				err: func(s string) error {
					return directives.Error{
						Message: "while parsing `include` statement",
						Range:   Range{End: 4, Text: s},
						Wrapped: directives.Error{
							Range:   directives.Range{End: 4, Text: s},
							Message: `while reading "include"`,
						},
					}
				},
			},
			{
				text: `include "foo\n`,
				want: func(s string) directives.Include {
					return directives.Include{
						Range: Range{End: 14, Text: s},
						IncludePath: directives.QuotedString{
							Range:   Range{Start: 8, End: 14, Text: s},
							Content: Range{Start: 9, End: 14, Text: s},
						},
					}
				},
				err: func(s string) error {
					return directives.Error{
						Message: "while parsing `include` statement",
						Range:   Range{End: 14, Text: s},
						Wrapped: directives.Error{
							Range:   directives.Range{Start: 8, End: 14, Text: s},
							Message: `while parsing quoted string`,
							Wrapped: directives.Error{
								Message: "unexpected end of file, want `\"`",
								Range:   Range{Start: 14, End: 14, Text: s},
							},
						},
					}
				},
			},
		},
		desc: "p.parseInclude()",
		fn: func(p *Parser) (directives.Include, error) {
			return p.parseInclude()
		},
	}.run(t)
}

func TestParseQuotedString(t *testing.T) {
	parserTest[directives.QuotedString]{
		desc: "p.parseQuotedString()",
		fn:   func(p *Parser) (directives.QuotedString, error) { return p.parseQuotedString() },
		tests: []testcase[directives.QuotedString]{
			{
				text: "\"\"",
				want: func(s string) directives.QuotedString {
					return directives.QuotedString{
						Range:   Range{End: 2, Text: s},
						Content: Range{Start: 1, End: 1, Text: s},
					}
				},
			},
			{
				text: "\"foo",
				want: func(s string) directives.QuotedString {
					return directives.QuotedString{
						Range:   Range{End: 4, Text: s},
						Content: Range{Start: 1, End: 4, Text: s},
					}
				},
				err: func(s string) error {
					return directives.Error{
						Message: "while parsing quoted string",
						Range:   Range{End: 4, Text: s},
						Wrapped: directives.Error{
							Range:   Range{Start: 4, End: 4, Text: s},
							Message: "unexpected end of file, want `\"`",
						},
					}
				},
			},
			{
				text: "\"foo\"",
				want: func(s string) directives.QuotedString {
					return directives.QuotedString{
						Range:   Range{End: 5, Text: s},
						Content: Range{Start: 1, End: 4, Text: s},
					}
				},
			},
			{
				text: "foo",
				want: func(s string) directives.QuotedString {
					return directives.QuotedString{Range: Range{Text: s}}
				},
				err: func(s string) error {
					return directives.Error{
						Message: "while parsing quoted string",
						Range:   Range{Text: s},
						Wrapped: directives.Error{
							Range:   Range{Text: s},
							Message: "unexpected character `f`, want `\"`",
						},
					}
				},
			},
		},
	}.run(t)
}

func TestParseTransaction(t *testing.T) {
	parserTest[directives.Transaction]{
		tests: []testcase[directives.Transaction]{
			{
				text: "\"foo\"\n" + "A B 1 CHF\n", // 6 + 10
				want: func(t string) directives.Transaction {
					return directives.Transaction{
						Range: Range{End: 16, Text: t},
						Description: directives.QuotedString{
							Range:   Range{End: 5, Text: t},
							Content: Range{Start: 1, End: 4, Text: t},
						},
						Bookings: []directives.Booking{
							{
								Range:     Range{Start: 6, End: 15, Text: t},
								Credit:    directives.Account{Range: Range{Start: 6, End: 7, Text: t}},
								Debit:     directives.Account{Range: Range{Start: 8, End: 9, Text: t}},
								Amount:    directives.Decimal{Range: Range{Start: 10, End: 11, Text: t}},
								Commodity: directives.Commodity{Range: Range{Start: 12, End: 15, Text: t}},
							},
						},
					}
				},
			},
			{
				text: "\"foo\"\n" + "A B 1 CHF\n" + "B A 1 CHF\n", // 6 + 10 + 10
				want: func(t string) directives.Transaction {
					return directives.Transaction{
						Range: Range{End: 26, Text: t},
						Description: directives.QuotedString{
							Range:   Range{End: 5, Text: t},
							Content: Range{Start: 1, End: 4, Text: t},
						},
						Bookings: []directives.Booking{
							{
								Range:     Range{Start: 6, End: 15, Text: t},
								Credit:    directives.Account{Range: Range{Start: 6, End: 7, Text: t}},
								Debit:     directives.Account{Range: Range{Start: 8, End: 9, Text: t}},
								Amount:    directives.Decimal{Range: Range{Start: 10, End: 11, Text: t}},
								Commodity: directives.Commodity{Range: Range{Start: 12, End: 15, Text: t}},
							},
							{
								Range:     Range{Start: 16, End: 25, Text: t},
								Credit:    directives.Account{Range: Range{Start: 16, End: 17, Text: t}},
								Debit:     directives.Account{Range: Range{Start: 18, End: 19, Text: t}},
								Amount:    directives.Decimal{Range: Range{Start: 20, End: 21, Text: t}},
								Commodity: directives.Commodity{Range: Range{Start: 22, End: 25, Text: t}},
							},
						},
					}
				},
			},
			{
				text: "\"foo\"\n" + "A B 1 CHF", // 6 + 10
				want: func(t string) directives.Transaction {
					return directives.Transaction{
						Range: Range{End: 15, Text: t},
						Description: directives.QuotedString{
							Range:   Range{End: 5, Text: t},
							Content: Range{Start: 1, End: 4, Text: t},
						},
						Bookings: []directives.Booking{
							{
								Range:     Range{Start: 6, End: 15, Text: t},
								Credit:    directives.Account{Range: Range{Start: 6, End: 7, Text: t}},
								Debit:     directives.Account{Range: Range{Start: 8, End: 9, Text: t}},
								Amount:    directives.Decimal{Range: Range{Start: 10, End: 11, Text: t}},
								Commodity: directives.Commodity{Range: Range{Start: 12, End: 15, Text: t}},
							},
						},
					}
				},
			},
			{
				text: strings.Join([]string{`"foo"`, "A B"}, "\n"), // 6 + 10
				want: func(t string) directives.Transaction {
					return directives.Transaction{
						Range: Range{End: 9, Text: t},
						Description: directives.QuotedString{
							Range:   Range{End: 5, Text: t},
							Content: Range{Start: 1, End: 4, Text: t},
						},
						Bookings: []directives.Booking{
							{
								Range:  Range{Start: 6, End: 9, Text: t},
								Credit: directives.Account{Range: Range{Start: 6, End: 7, Text: t}},
								Debit:  directives.Account{Range: Range{Start: 8, End: 9, Text: t}},
							},
						},
					}
				},
				err: func(s string) error {
					return directives.Error{
						Message: "while parsing transaction",
						Range:   Range{End: 9, Text: s},
						Wrapped: directives.Error{
							Range:   directives.Range{Start: 6, End: 9, Text: s},
							Message: "while parsing booking",
							Wrapped: directives.Error{
								Range:   directives.Range{Start: 9, End: 9, Text: s},
								Message: "unexpected end of file, want whitespace",
							},
						},
					}
				},
			},
		},
		desc: "p.parseTransaction()",
		fn: func(p *Parser) (directives.Transaction, error) {
			return p.parseTransaction(directives.Date{}, directives.Addons{})
		},
	}.run(t)
}

func TestParseDirective(t *testing.T) {
	parserTest[directives.Directive]{
		tests: []testcase[directives.Directive]{
			{
				text: "@performance(USD)\n" + "2023-04-03 \"foo\"\n" + "A B 1 CHF\n", // 18 + 17 + 10
				want: func(s string) directives.Directive {
					return directives.Directive{
						Range: Range{End: 45, Text: s},

						Directive: directives.Transaction{
							Range: Range{End: 45, Text: s},
							Date:  directives.Date{Range: directives.Range{Start: 18, End: 28, Text: s}},
							Description: directives.QuotedString{
								Range:   Range{Start: 29, End: 34, Text: s},
								Content: Range{Start: 30, End: 33, Text: s},
							},
							Bookings: []directives.Booking{
								{
									Range:     Range{Start: 35, End: 44, Text: s},
									Credit:    directives.Account{Range: Range{Start: 35, End: 36, Text: s}},
									Debit:     directives.Account{Range: Range{Start: 37, End: 38, Text: s}},
									Amount:    directives.Decimal{Range: Range{Start: 39, End: 40, Text: s}},
									Commodity: directives.Commodity{Range: Range{Start: 41, End: 44, Text: s}},
								},
							},
							Addons: directives.Addons{
								Range: Range{End: 18, Text: s},
								Performance: directives.Performance{
									Range: Range{End: 17, Text: s},
									Targets: []directives.Commodity{
										{Range: Range{Start: 13, End: 16, Text: s}},
									},
								},
							},
						},
					}
				},
			},
			{
				text: "2023-04-03 \"foo\"\n" + "A B 1 CHF\n", // 17 + 10
				want: func(s string) directives.Directive {
					return directives.Directive{
						Range: Range{End: 27, Text: s},

						Directive: directives.Transaction{
							Range: Range{End: 27, Text: s},
							Date:  directives.Date{Range: directives.Range{End: 10, Text: s}},
							Description: directives.QuotedString{
								Range:   Range{Start: 11, End: 16, Text: s},
								Content: Range{Start: 12, End: 15, Text: s},
							},
							Bookings: []directives.Booking{
								{
									Range:     Range{Start: 17, End: 26, Text: s},
									Credit:    directives.Account{Range: Range{Start: 17, End: 18, Text: s}},
									Debit:     directives.Account{Range: Range{Start: 19, End: 20, Text: s}},
									Amount:    directives.Decimal{Range: Range{Start: 21, End: 22, Text: s}},
									Commodity: directives.Commodity{Range: Range{Start: 23, End: 26, Text: s}},
								},
							},
						},
					}
				},
			},
			{
				text: " 2023-04-03 \"foo\"\n" + "A B 1 CHF\n", // 17 + 10
				want: func(s string) directives.Directive {
					return directives.Directive{
						Range: Range{Text: s},
					}
				},
				err: func(s string) error {
					return directives.Error{
						Message: "while parsing directive",
						Range:   Range{Text: s},
						Wrapped: directives.Error{
							Range:   Range{Text: s},
							Message: "while parsing the date",
							Wrapped: directives.Error{
								Range:   Range{Text: s},
								Message: "unexpected character ` `, want a digit",
							},
						},
					}
				},
			},
			{
				text: "2023-04-03 \"foo",
				want: func(s string) directives.Directive {
					return directives.Directive{
						Range: Range{End: 15, Text: s},
						Directive: directives.Transaction{
							Range: Range{End: 15, Text: s},
							Date:  directives.Date{Range: directives.Range{End: 10, Text: s}},
							Description: directives.QuotedString{
								Range:   directives.Range{Start: 11, End: 15, Text: s},
								Content: directives.Range{Start: 12, End: 15, Text: s},
							},
						},
					}
				},
				err: func(s string) error {
					return directives.Error{
						Range:   directives.Range{End: 15, Text: s},
						Message: "while parsing directive",
						Wrapped: directives.Error{
							Range:   directives.Range{End: 15, Text: s},
							Message: "while parsing transaction",
							Wrapped: directives.Error{
								Range:   directives.Range{Start: 11, End: 15, Text: s},
								Message: "while parsing quoted string",
								Wrapped: directives.Error{
									Range:   directives.Range{Start: 15, End: 15, Text: s},
									Message: "unexpected end of file, want `\"`",
								},
							},
						},
					}
				},
			},
			{
				text: "2023-04-03 open B:A",
				want: func(s string) directives.Directive {
					return directives.Directive{
						Range: Range{End: 19, Text: s},
						Directive: directives.Open{
							Range:   Range{End: 19, Text: s},
							Date:    directives.Date{Range: directives.Range{End: 10, Text: s}},
							Account: directives.Account{Range: directives.Range{Start: 16, End: 19, Text: s}},
						},
					}
				},
			},
			{
				text: `include "foo/foo.knut"`,
				want: func(s string) directives.Directive {
					return directives.Directive{
						Range: Range{End: 22, Text: s},
						Directive: directives.Include{
							Range: Range{End: 22, Text: s},
							IncludePath: directives.QuotedString{
								Range:   Range{Start: 8, End: 22, Text: s},
								Content: Range{Start: 9, End: 21, Text: s},
							},
						},
					}
				},
			},
			{
				text: "2023-04-03 close B:A",
				want: func(s string) directives.Directive {
					return directives.Directive{
						Range: Range{End: 20, Text: s},
						Directive: directives.Close{
							Range:   Range{End: 20, Text: s},
							Date:    directives.Date{Range: directives.Range{End: 10, Text: s}},
							Account: directives.Account{Range: directives.Range{Start: 17, End: 20, Text: s}},
						},
					}
				},
			},
			{
				text: "2023-04-03 balance B:A 1 USD",
				want: func(s string) directives.Directive {
					return directives.Directive{
						Range: Range{End: 28, Text: s},
						Directive: directives.Assertion{
							Range:     Range{End: 28, Text: s},
							Date:      directives.Date{Range: directives.Range{End: 10, Text: s}},
							Account:   directives.Account{Range: directives.Range{Start: 19, End: 22, Text: s}},
							Amount:    directives.Decimal{Range: directives.Range{Start: 23, End: 24, Text: s}},
							Commodity: directives.Commodity{Range: Range{Start: 25, End: 28, Text: s}},
						},
					}
				},
			},
			{
				text: "2023-04-03 price CHF 0.83 USD",
				want: func(s string) directives.Directive {
					return directives.Directive{
						Range: Range{End: 29, Text: s},
						Directive: directives.Price{
							Range:     Range{End: 29, Text: s},
							Date:      directives.Date{Range: directives.Range{End: 10, Text: s}},
							Commodity: directives.Commodity{Range: directives.Range{Start: 17, End: 20, Text: s}},
							Price:     directives.Decimal{Range: directives.Range{Start: 21, End: 25, Text: s}},
							Target:    directives.Commodity{Range: Range{Start: 26, End: 29, Text: s}},
						},
					}
				},
			},
		},
		desc: "p.parseDirective()",
		fn: func(p *Parser) (directives.Directive, error) {
			return p.parseDirective()
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
					return Range{End: 1, Text: s}
				},
			},
			{
				text: " \n",
				want: func(s string) Range {
					return Range{End: 2, Text: s}
				},
			},
			{
				text: " foo",
				want: func(s string) Range {
					return Range{End: 1, Text: s}
				},
				err: func(s string) error {
					return directives.Error{
						Message: "while reading the rest of the line",
						Range:   Range{End: 1, Text: s},
						Wrapped: directives.Error{
							Message: "unexpected character `f`, want `\n`",
							Range:   Range{Start: 1, End: 1, Text: s},
						},
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
					return Range{Text: s}
				},
			},
			{
				text: " \n",
				want: func(s string) Range {
					return Range{End: 1, Text: s}
				},
			},
			{
				text: " foo",
				want: func(s string) Range {
					return Range{End: 1, Text: s}
				},
			},
			{
				text: "   foo",
				want: func(s string) Range {
					return Range{End: 3, Text: s}
				},
			},
			{
				text: "foo",
				want: func(s string) Range {
					return Range{Text: s}
				},
				err: func(s string) error {
					return directives.Error{
						Message: "unexpected character `f`, want whitespace or a newline",
						Range:   Range{Text: s},
					}
				},
			},
		},
	}.run(t)
}
