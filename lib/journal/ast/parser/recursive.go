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

package parser

import (
	"path"
	"path/filepath"
	"sync"

	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
)

// RecursiveParser parses a file hierarchy recursively.
type RecursiveParser struct {
	File    string
	Context journal.Context
}

// Parse parses the journal at the path, and branches out for include files
func (j *RecursiveParser) Parse() chan interface{} {
	var (
		ch = make(chan interface{}, 100)
		wg sync.WaitGroup
	)

	wg.Add(1)
	go j.parseRecursively(&wg, ch, j.File)

	// Parse and eventually close input channel
	go func() {
		defer close(ch)
		wg.Wait()
	}()
	return ch
}

func (j *RecursiveParser) parseRecursively(wg *sync.WaitGroup, ch chan<- interface{}, file string) {
	defer wg.Done()
	p, cls, err := FromPath(j.Context, file)
	if err != nil {
		ch <- err
		return
	}
	defer func() {
		err = cls()
		if err != nil {
			ch <- err
		}
	}()
	for d := range p.ParseAll() {
		switch t := d.(type) {

		case error:
			ch <- t
			return

		case *ast.Include:
			wg.Add(1)
			go j.parseRecursively(wg, ch, path.Join(filepath.Dir(file), t.Path))

		default:
			ch <- d
		}
	}
}
