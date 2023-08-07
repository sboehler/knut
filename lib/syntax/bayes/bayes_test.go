package bayes

import (
	"bytes"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sboehler/knut/lib/syntax"
	"github.com/sboehler/knut/lib/syntax/parser"
)

func TestPrintFile(t *testing.T) {
	tests := []struct {
		desc     string
		training string
		target   string
		want     string
	}{
		{
			desc: "print transaction",
			training: lines(
				`2022-03-03 "Hello world"`,
				`A B 400 CHF`,
				``,
				`2022-03-03 "Hello Europe"`,
				`A C 400 CHF`,
				``,
				`2022-03-03 "Hello Asia"`,
				`A D 400 CHF`,
				``,
			),
			target: lines(
				`2022-03-03 "hello europe"`,
				`A TBD 400 CHF`,
				``,
				`2022-03-03 "hello world"`,
				`A TBD 400 CHF`,
				``,
				`2022-03-03 "hello asia"`,
				`A TBD 400 CHF`,
			),
			want: lines(
				`2022-03-03 "hello europe"`,
				`A C        400 CHF`,
				``,
				`2022-03-03 "hello world"`,
				`A B        400 CHF`,
				``,
				`2022-03-03 "hello asia"`,
				`A D        400 CHF`,
			),
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			training := parse(t, test.training)
			target := parse(t, test.target)
			model := NewModel("TBD")
			for _, d := range training.Directives {
				if t, ok := d.Directive.(syntax.Transaction); ok {
					model.Update(&t)
				}
			}

			for _, d := range target.Directives {
				if t, ok := d.Directive.(syntax.Transaction); ok {
					model.Infer(&t)
				}
			}
			var got bytes.Buffer

			err := syntax.FormatFile(&got, target)

			if err != nil {
				t.Fatalf("pr.Format() returned unexpected error: %v", err)
			}
			if diff := cmp.Diff(test.want, got.String()); diff != "" {
				t.Fatalf("PrintFile() returned unexpected diff (-want/+got):\n%s\n", diff)
			}
		})
	}
}

func lines(ss ...string) string {
	return strings.Join(ss, "\n") + "\n"
}

func parse(t *testing.T, s string) syntax.File {
	t.Helper()
	p := parser.New(s, "")
	if err := p.Advance(); err != nil {
		t.Fatal(err)
	}
	f, err := p.ParseFile()
	if err != nil {
		t.Fatalf("p.ParseFile() returned unexpected error: %#v", err)
	}
	return f
}
