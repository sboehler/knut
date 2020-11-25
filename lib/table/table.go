// Copyright 2020 Silvio Böhler
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
	"io"
	"unicode/utf8"
)

// CellType is the type of a table cell.
type CellType int

// Table is a matrix of table cells.
type Table struct {
	columns []int
	rows    []*Row
}

// New creates a new table with column groups.
func New(groups ...int) *Table {
	var columns []int
	for groupNo, groupSize := range groups {
		for i := 0; i < groupSize; i++ {
			columns = append(columns, groupNo)
		}
	}
	return &Table{columns: columns}
}

// Width returns the width of this table.
func (t *Table) Width() int {
	return len(t.columns)
}

// AddRow adds a row.
func (t *Table) AddRow() *Row {
	row := &Row{cells: make([]cell, 0, t.Width())}
	t.rows = append(t.rows, row)
	return row
}

// AddSeparatorRow adds a separator row.
func (t *Table) AddSeparatorRow() {
	c := t.AddRow()
	for i := 0; i < t.Width(); i++ {
		c.addCell(SeparatorCell{})
	}
}

// AddEmptyRow adds a separator row.
func (t *Table) AddEmptyRow() {
	c := t.AddRow()
	for i := 0; i < t.Width(); i++ {
		c.addCell(emptyCell{})
	}
}

// Render renders this table to a string.
func (t *Table) Render(w io.StringWriter) {
	widths := make([]int, t.Width())
	for _, r := range t.rows {
		for i, c := range r.cells {
			if widths[i] < c.length() {
				widths[i] = c.length()
			}
		}
	}
	groups := map[int]int{}
	for i, w := range widths {
		if groups[t.columns[i]] < w {
			groups[t.columns[i]] = w
		}
	}
	for i, w := range widths {
		if w < groups[i] {
			widths[i] = groups[i]
		}
	}
	for _, r := range t.rows {
		if r.cells[0].isSep() {
			w.WriteString("+-")
		} else {
			w.WriteString("| ")
		}

		for i, c := range r.cells {
			c.render(widths[i], w)
			if i < len(r.cells)-1 {
				w.WriteString(createSep(c, r.cells[i+1]))
			}
		}
		if r.cells[len(r.cells)-1].isSep() {
			w.WriteString("-+\n")
		} else {
			w.WriteString(" |\n")
		}
	}
	w.WriteString("\n")
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

// Row is a table row.
type Row struct {
	cells []cell
}

func (r *Row) addCell(c cell) {
	r.cells = append(r.cells, c)
}

// AddEmpty adds an empty cell.
func (r *Row) AddEmpty() *Row {
	r.addCell(emptyCell{})
	return r
}

// AddText adds a text cell.
func (r *Row) AddText(content string, align Alignment) *Row {
	r.addCell(textCell{
		content,
		align,
	})
	return r
}

// AddIndented adds an indented cell.
func (r *Row) AddIndented(content string, indent int) *Row {
	r.addCell(indentedCell{content, indent})
	return r
}

type cell interface {
	length() int
	render(int, io.StringWriter)
	isSep() bool
}

// Alignment is the alignment of a table cell.
type Alignment int

const (
	// Left aligns to the left.
	Left Alignment = iota
	// Right align to the right.
	Right
	// Center centers.
	Center
)

// indentedCell is a cell with an indent.
type indentedCell struct {
	Content string
	Indent  int
}

func (t indentedCell) length() int {
	return t.Indent + utf8.RuneCountInString(t.Content)
}

func (t indentedCell) render(l int, b io.StringWriter) {
	for i := 0; i < t.Indent; i++ {
		b.WriteString(" ")
	}
	b.WriteString(t.Content)
	for i := 0; i < l-utf8.RuneCountInString(t.Content)-t.Indent; i++ {
		b.WriteString(" ")
	}
}

func (t indentedCell) isSep() bool {
	return false
}

// textCell is a cell containing text.
type textCell struct {
	Content string
	Align   Alignment
}

func (t textCell) length() int {
	return utf8.RuneCountInString(t.Content)
}

func (t textCell) render(l int, s io.StringWriter) {
	var before int
	switch t.Align {
	case Left:
		before = 0
	case Right:
		before = l - utf8.RuneCountInString(t.Content)
	case Center:
		before = (l - utf8.RuneCountInString(t.Content)) / 2
	}

	for i := 0; i < before; i++ {
		s.WriteString(" ")
	}
	s.WriteString(t.Content)
	for i := 0; i < l-before-utf8.RuneCountInString(t.Content); i++ {
		s.WriteString(" ")
	}
}

func (t textCell) isSep() bool {
	return false
}

// SeparatorCell is a cell containing a separator.
type SeparatorCell struct{}

func (s SeparatorCell) length() int {
	return 0
}

func (SeparatorCell) render(l int, s io.StringWriter) {
	for i := 0; i < l; i++ {
		s.WriteString("-")
	}
}
func (SeparatorCell) isSep() bool {
	return true
}

// emptyCell is an empty cell.
type emptyCell struct{}

func (emptyCell) length() int {
	return 0
}

func (emptyCell) render(l int, s io.StringWriter) {
	for i := 0; i < l; i++ {
		s.WriteString(" ")
	}
}

func (emptyCell) isSep() bool {
	return false
}
