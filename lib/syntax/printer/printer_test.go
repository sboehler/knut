package printer

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/sboehler/knut/lib/syntax/parser"
)

func TestPrintAccount(t *testing.T) {
	tests := []struct {
		desc string
		text string
		want string
	}{
		{
			desc: "print transaction",
			text: join(
				`2022-03-03    "Hello, world"`,
				`A:B:C       C:B:ASDF   400 CHF   `,
			),
			want: join(
				`2022-03-03 "Hello, world"`,
				"A:B:C C:B:ASDF        400 CHF",
			),
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			p := parser.New(test.text, "")
			p.Advance()
			f, err := p.ParseFile()
			fmt.Println(test.text)
			if err != nil {
				t.Errorf("%#v", err)
			}
			pr := NewPrinter()
			var got strings.Builder

			_, err = pr.PrintFile(&got, f)

			if diff := cmp.Diff(err, nil, cmpopts.EquateErrors()); diff != "" {
				t.Fatalf("PrintFile() error returned unexpected diff (-want/+got):\n%s\n", diff)
			}
			if diff := cmp.Diff(test.want, got.String()); diff != "" {
				t.Fatalf("PrintFile() returned unexpected diff (-want/+got):\n%s\n", diff)
			}
		})
	}
}

func join(ss ...string) string {
	return strings.Join(ss, "\n")
}
