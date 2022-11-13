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

package table

import (
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/fatih/color"
	"github.com/shopspring/decimal"
)

// TextRenderer renders a table to text.
type TextRenderer struct {
	table     *Table
	Color     bool
	Thousands bool
	Round     int32
}

var (
	green = color.New(color.FgGreen)
	red   = color.New(color.FgRed)
)

// Render renders this table to a string.
func (r *TextRenderer) Render(t *Table, w io.Writer) error {
	r.table = t
	color.NoColor = !r.Color

	widths := make([]int, r.table.Width())
	for _, row := range r.table.rows {
		for i, c := range row.cells {
			if widths[i] < r.minLengthCell(c) {
				widths[i] = r.minLengthCell(c)
			}
		}
	}
	groups := make(map[int]int)
	for i, w := range widths {
		if groups[r.table.columns[i]] < w {
			groups[r.table.columns[i]] = w
		}
	}
	for i, w := range widths {
		if w < groups[i] {
			widths[i] = groups[i]
		}
	}
	for _, row := range r.table.rows {
		if row.cells[0].isSep() {
			if _, err := io.WriteString(w, "+-"); err != nil {
				return err
			}
		} else {
			if _, err := io.WriteString(w, "| "); err != nil {
				return err
			}
		}

		for i, c := range row.cells {
			r.renderCell(c, widths[i], w)
			if i < len(row.cells)-1 {
				if _, err := io.WriteString(w, createSep(c, row.cells[i+1])); err != nil {
					return err
				}
			}
		}
		if row.cells[len(row.cells)-1].isSep() {
			if _, err := io.WriteString(w, "-+\n"); err != nil {
				return err
			}
		} else {
			if _, err := io.WriteString(w, " |\n"); err != nil {
				return err
			}
		}
	}
	_, err := io.WriteString(w, "\n")
	r.table = nil
	return err
}

func (r *TextRenderer) renderCell(c cell, l int, w io.Writer) error {
	switch t := c.(type) {

	case emptyCell:
		return writeSpace(w, l)

	case SeparatorCell:
		return writeStrings(w, "-", l)

	case textCell:
		var before int
		switch t.Align {
		case Left:
			before = t.Indent
		case Right:
			before = l - utf8.RuneCountInString(t.Content)
		case Center:
			before = (l - utf8.RuneCountInString(t.Content)) / 2
		}
		if err := writeSpace(w, before); err != nil {
			return err
		}
		if err := writeString(w, t.Content); err != nil {
			return err
		}
		return writeSpace(w, l-before-utf8.RuneCountInString(t.Content))

	case numberCell:
		var (
			s      = r.numToString(t.n)
			before = l - utf8.RuneCountInString(s)
		)
		if err := writeSpace(w, before); err != nil {
			return err
		}
		var err error
		switch {
		case t.n.LessThan(decimal.Zero):
			_, err = red.Fprint(w, s)
		case t.n.Equal(decimal.Zero):
			_, err = fmt.Fprint(w, s)
		case t.n.GreaterThan(decimal.Zero):
			_, err = green.Fprint(w, s)
		}
		if err != nil {
			return err
		}
		return writeSpace(w, l-before-utf8.RuneCountInString(s))
	}
	return fmt.Errorf("%v is not a valid cell type", c)
}

func writeStrings(w io.Writer, s string, l int) error {
	for i := 0; i < l; i++ {
		if err := writeString(w, s); err != nil {
			return err
		}
	}
	return nil
}

func writeString(w io.Writer, s string) error {
	_, err := io.WriteString(w, s)
	return err
}

func writeSpace(w io.Writer, l int) error {
	return writeStrings(w, " ", l)
}

func (r *TextRenderer) minLengthCell(c cell) int {
	switch t := c.(type) {
	case emptyCell, SeparatorCell:
		return 0
	case textCell:
		if t.Align == Left {
			return t.Indent + utf8.RuneCountInString(t.Content)
		}
		return utf8.RuneCountInString(t.Content)
	case numberCell:
		return utf8.RuneCountInString(r.numToString(t.n))
	}
	return 0
}

func createSep(c1, c2 cell) string {
	switch {
	case c1.isSep() && c2.isSep():
		return "-+-"
	case c1.isSep():
		return "-+ "
	case c2.isSep():
		return " +-"
	default:
		return " | "
	}
}

var k = decimal.RequireFromString("1000")

func (r *TextRenderer) numToString(d decimal.Decimal) string {
	if r.Thousands {
		d = d.Div(k)
	}
	return addThousandsSep(d.StringFixed(r.Round))
}

func addThousandsSep(e string) string {
	index := strings.Index(e, ".")
	if index < 0 {
		index = len(e)
	}
	var (
		b  strings.Builder
		ok bool
	)
	for i, ch := range e {
		if i >= index && ch != '-' {
			b.WriteString(e[i:])
			break
		}
		if (index-i)%3 == 0 && ok {
			b.WriteRune(',')
		}
		b.WriteRune(ch)
		if unicode.IsDigit(ch) {
			ok = true
		}
	}
	return b.String()
}
