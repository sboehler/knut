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
	"encoding/csv"
	"fmt"
	"io"
)

// CSVRenderer renders a table to text.
type CSVRenderer struct{}

// Render renders this table to a string.
func (r *CSVRenderer) Render(t *Table, w io.Writer) error {
	writer := csv.NewWriter(w)
	for _, row := range t.rows {
		var rec []string
		for _, c := range row.cells {
			s, err := r.renderCell(c)
			if err != nil {
				return err
			}
			rec = append(rec, s)
		}
		var hasText bool
		for _, r := range rec {
			if len(r) > 0 {
				hasText = true
				break
			}
		}
		if !hasText {
			continue
		}
		if err := writer.Write(rec); err != nil {
			return err
		}
	}
	writer.Flush()
	return nil
}

func (r *CSVRenderer) renderCell(c cell) (string, error) {
	switch t := c.(type) {

	case emptyCell, SeparatorCell:
		return "", nil

	case textCell:
		return t.Content, nil

	case numberCell:
		return t.n.String(), nil

	case percentCell:
		return fmt.Sprintf("%f", t.n), nil
	}
	return "", fmt.Errorf("%v is not a valid cell type", c)
}
