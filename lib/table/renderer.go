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

// Renderer renders a table to text.
type Renderer struct {
	Table      *Table
	Color      bool
	Thousands  bool
	Round      int32
	green, red *color.Color
}

// NewConsoleRenderer returns a new console renderer.
func NewConsoleRenderer(t *Table, enableColor bool, thousands bool, round int32) *Renderer {
	return &Renderer{
		Table:     t,
		Color:     enableColor,
		Thousands: thousands,
		Round:     round,
		green:     color.New(color.FgGreen),
		red:       color.New(color.FgRed),
	}
}

// Render renders this table to a string.
func (r *Renderer) Render(w io.Writer) error {
	color.NoColor = !r.Color

	widths := make([]int, r.Table.Width())
	for _, row := range r.Table.rows {
		for i, c := range row.cells {
			if widths[i] < r.minLengthCell(c) {
				widths[i] = r.minLengthCell(c)
			}
		}
	}
	groups := map[int]int{}
	for i, w := range widths {
		if groups[r.Table.columns[i]] < w {
			groups[r.Table.columns[i]] = w
		}
	}
	for i, w := range widths {
		if w < groups[i] {
			widths[i] = groups[i]
		}
	}
	for _, row := range r.Table.rows {
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
	return err
}

func (r *Renderer) renderCell(c cell, l int, w io.Writer) error {
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
		s := r.numToString(t.n)
		var before = l - utf8.RuneCountInString(s)
		if err := writeSpace(w, before); err != nil {
			return err
		}
		var err error
		switch {
		case t.n.LessThan(decimal.Zero):
			_, err = r.red.Fprint(w, s)
		case t.n.Equal(decimal.Zero):
			_, err = fmt.Fprint(w, s)
		case t.n.GreaterThan(decimal.Zero):
			_, err = r.green.Fprint(w, s)
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

func (r *Renderer) minLengthCell(c cell) int {
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

func (r *Renderer) numToString(d decimal.Decimal) string {
	if r.Thousands {
		d = d.Div(k)
	}
	e := d.StringFixed(r.Round)
	return addThousandsSep(e)
}

func addThousandsSep(e string) string {
	index := strings.Index(e, ".")
	if index < 0 {
		index = len(e)
	}
	b := strings.Builder{}
	ok := false
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
