package printer

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/sboehler/knut/lib/syntax/parser"
)

func TestPrintFile(t *testing.T) {
	tests := []struct {
		desc string
		text string
		want string
	}{
		{
			desc: "print transaction",
			text: lines(
				`2022-03-03    "Hello, world"`,
				`A:B:C       C:B:ASDF   400 CHF   `,
			),
			want: lines(
				`2022-03-03 "Hello, world"`,
				"A:B:C C:B:ASDF        400 CHF",
				"",
			),
		},
		{
			desc: "print transactions",
			text: lines(
				`2022-03-03    "Hello, world"`,
				`A:B:C       C:B:ASDF   400 CHF   `,
				``,
				`2023-03-03    "Hello, world"`,
				`A:B:C       C:B:ASDF   400 CHF   `,
			),
			want: lines(
				`2022-03-03 "Hello, world"`,
				"A:B:C C:B:ASDF        400 CHF",
				``,
				`2023-03-03 "Hello, world"`,
				"A:B:C C:B:ASDF        400 CHF",
				"",
			),
		},
		{
			desc: "print transactions with addons",
			text: lines(
				`@performance(    USD , EUR  )`,
				`2022-03-03    "Hello, world"`,
				`A:B:C       C:B:ASDF   400 CHF   `,
				``,
				`@accrue    monthly   2023-01-01    2023-12-01    Assets:Receivables   `,
				`2023-03-03    "Hello, world"`,
				`A:B:C       C:B:ASDF   400 CHF   `,
			),
			want: lines(
				`@performance(USD,EUR)`,
				`2022-03-03 "Hello, world"`,
				"A:B:C C:B:ASDF        400 CHF",
				``,
				"@accrue monthly 2023-01-01 2023-12-01 Assets:Receivables",
				`2023-03-03 "Hello, world"`,
				"A:B:C C:B:ASDF        400 CHF",
				"",
			),
		},
		{
			desc: "include",
			text: lines(
				`include     "foo"   `,
			),
			want: lines(
				`include "foo"`,
			),
		},
		{
			desc: "print includes",
			text: lines(
				`include "foo1"     `,
				`include    "foo2"      `,
				`include         "foo3" `,
			),
			want: lines(
				`include "foo1"`,
				`include "foo2"`,
				`include "foo3"`,
			),
		},
		{
			desc: "print open",
			text: lines(
				`2022-03-03       open XYZ:ABC`,
			),
			want: lines(
				`2022-03-03 open XYZ:ABC`,
			),
		},
		{
			desc: "print opens",
			text: lines(
				`2022-03-03       open XYZ:ABC1     `,
				`2022-03-03    open XYZ:ABC2      `,
				`2022-03-03       open         XYZ:ABC3`,
			),
			want: lines(
				`2022-03-03 open XYZ:ABC1`,
				`2022-03-03 open XYZ:ABC2`,
				`2022-03-03 open XYZ:ABC3`,
			),
		},
		{
			desc: "print close",
			text: lines(
				`2022-03-03       close XYZ:ABC`,
			),
			want: lines(
				`2022-03-03 close XYZ:ABC`,
			),
		},
		{
			desc: "print closes",
			text: lines(
				`2022-03-03       close XYZ:ABC1     `,
				`2022-03-03    close XYZ:ABC2      `,
				`2022-03-03       close         XYZ:ABC3`,
			),
			want: lines(
				`2022-03-03 close XYZ:ABC1`,
				`2022-03-03 close XYZ:ABC2`,
				`2022-03-03 close XYZ:ABC3`,
			),
		},
		{
			desc: "print assertion",
			text: lines(`2022-03-03  balance    XYZ:ABC -80.23 CHF`),
			want: lines(`2022-03-03 balance XYZ:ABC -80.23 CHF`),
		},
		{
			desc: "print assertions",
			text: lines(
				`2022-03-03  balance    XYZ:ABC:1      -80.23 CHF`,
				`2022-03-03  balance    XYZ:ABC:2   80      CHF`,
				`2022-03-03  balance    XYZ:ABC:3  -0.3             CHF`,
			),
			want: lines(
				`2022-03-03 balance XYZ:ABC:1 -80.23 CHF`,
				`2022-03-03 balance XYZ:ABC:2 80 CHF`,
				`2022-03-03 balance XYZ:ABC:3 -0.3 CHF`,
			),
		},
		{
			desc: "print price",
			text: lines(
				`2022-03-03  price   USD  0.894 CHF`,
			),
			want: lines(
				`2022-03-03 price USD 0.894 CHF`,
			),
		},
		{
			desc: "print prices",
			text: lines(
				`2022-03-03  price   USD  0.894 CHF`,
				`2022-03-03  price   USD  0.895 CHF`,
			),
			want: lines(
				`2022-03-03 price USD 0.894 CHF`,
				`2022-03-03 price USD 0.895 CHF`,
			),
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			p := parser.New(test.text, "")
			if err := p.Advance(); err != nil {
				t.Fatal(err)
			}
			f, err := p.ParseFile()
			if err != nil {
				t.Fatalf("p.ParseFile() returned unexpected error: %#v", err)
			}
			var got strings.Builder
			pr := Printer{Writer: &got}

			_, err = pr.PrintFile(f)

			if diff := cmp.Diff(err, nil, cmpopts.EquateErrors()); diff != "" {
				t.Fatalf("PrintFile() error returned unexpected diff (-want/+got):\n%s\n", diff)
			}
			if diff := cmp.Diff(test.want, got.String()); diff != "" {
				t.Fatalf("PrintFile() returned unexpected diff (-want/+got):\n%s\n", diff)
			}
		})
	}
}

func TestFormat(t *testing.T) {
	tests := []struct {
		desc string
		text string
		want string
	}{
		{
			desc: "print prices",
			text: lines(
				``,
				`// some prices`,
				`2022-03-03  price   USD      0.894 CHF`,
				`#comment`,
				`2022-03-03  price   USD  0.895 CHF`,
			),
			want: lines(
				``,
				`// some prices`,
				`2022-03-03 price USD 0.894 CHF`,
				`#comment`,
				`2022-03-03 price USD 0.895 CHF`,
			),
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			p := parser.New(test.text, "")
			if err := p.Advance(); err != nil {
				t.Fatal(err)
			}
			f, err := p.ParseFile()
			if err != nil {
				t.Fatalf("p.ParseFile() returned unexpected error: %#v", err)
			}
			var got strings.Builder
			pr := Printer{Writer: &got}

			err = pr.Format(f)

			if diff := cmp.Diff(err, nil, cmpopts.EquateErrors()); diff != "" {
				t.Fatalf("Format() error returned unexpected diff (-want/+got):\n%s\n", diff)
			}
			if diff := cmp.Diff(test.want, got.String()); diff != "" {
				t.Fatalf("Format() returned unexpected diff (-want/+got):\n%s\n", diff)
			}
		})
	}
}

func lines(ss ...string) string {
	return strings.Join(ss, "\n") + "\n"
}
