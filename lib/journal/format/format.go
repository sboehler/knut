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

package format

import (
	"io"
	"io/ioutil"

	"github.com/sboehler/knut/lib/journal"
)

type reader interface {
	io.RuneReader
	io.Reader
}

// Format formats the directives returned by p.
func Format(directives []journal.Directive, src reader, dest io.Writer) error {
	var (
		p          = journal.NewPrinter()
		srcBytePos int
	)
	p.Initialize(directives)
	for _, d := range directives {
		p0, p1 := d.Position().Start.BytePos, d.Position().End.BytePos

		// copy text before directive from src to dest
		if _, err := io.CopyN(dest, src, int64(p0-srcBytePos)); err != nil {
			return err
		}

		// seek forward over directive in src
		if _, err := ioutil.ReadAll(io.LimitReader(src, int64(p1-p0))); err != nil {
			return err
		}

		// write directive to dst
		if _, err := p.PrintDirective(dest, d); err != nil {
			return err
		}
		// update srcPos
		srcBytePos = p1
	}
	_, err := io.Copy(dest, src)
	return err
}
