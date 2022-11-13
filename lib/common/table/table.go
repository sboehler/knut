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
	"github.com/shopspring/decimal"
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
	var (
		cells = make([]cell, 0, t.Width())
		row   = &Row{cells}
	)
	t.rows = append(t.rows, row)
	return row
}

// AddSeparatorRow adds a separator row.
func (t *Table) AddSeparatorRow() {
	r := t.AddRow()
	for i := 0; i < t.Width(); i++ {
		r.addCell(SeparatorCell{})
	}
}

// AddEmptyRow adds a separator row.
func (t *Table) AddEmptyRow() {
	r := t.AddRow()
	for i := 0; i < t.Width(); i++ {
		r.addCell(emptyCell{})
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
		Indent:  0,
		Content: content,
		Align:   align,
	})
	return r
}

// AddNumber adds a number cell.
func (r *Row) AddNumber(n decimal.Decimal) *Row {
	r.addCell(numberCell{n})
	return r
}

// AddIndented adds an indented cell.
func (r *Row) AddIndented(content string, indent int) *Row {
	r.addCell(textCell{
		Content: content,
		Indent:  indent,
		Align:   Left,
	})
	return r
}

// FillEmpty fills the row with empty cells.
func (r *Row) FillEmpty() {
	for i := len(r.cells); i < cap(r.cells); i++ {
		r.AddEmpty()
	}
}

type cell interface {
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

// textCell is a cell containing text.
type textCell struct {
	Content string
	Align   Alignment
	Indent  int
}

func (t textCell) isSep() bool {
	return false
}

// textCell is a cell containing text.
type numberCell struct {
	n decimal.Decimal
}

func (t numberCell) isSep() bool {
	return false
}

// SeparatorCell is a cell containing a separator.
type SeparatorCell struct{}

func (SeparatorCell) isSep() bool {
	return true
}

// emptyCell is an empty cell.
type emptyCell struct{}

func (emptyCell) isSep() bool {
	return false
}
