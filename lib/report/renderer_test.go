package report

import "testing"

func TestFmt2(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"1000.000", "1,000.000"},
		{"1.234", "1.234"},
		{"12.34", "12.34"},
		{"123.45", "123.45"},
		{"1234.56", "1,234.56"},
		{"12345.67", "12,345.67"},
		{"12345678.9", "12,345,678.9"},
		{"12345678", "12,345,678"},
		{"-12345678", "-12,345,678"},
		{"-123.45", "-123.45"},
		{"0", "0"},
		{"10", "10"},
		{"100", "100"},
	}

	for _, test := range tests {
		test := test
		t.Run(test.input, func(t *testing.T) {
			got := addThousandsSep(test.input)
			if got != test.want {
				t.Errorf("fmt2(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}
