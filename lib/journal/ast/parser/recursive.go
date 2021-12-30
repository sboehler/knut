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
	"context"
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

	errCh chan error
	resCh chan ast.Directive

	wg sync.WaitGroup
}

// Parse parses the journal at the path, and branches out for include files
func (j *RecursiveParser) Parse(ctx context.Context) (<-chan ast.Directive, <-chan error) {
	j.resCh = make(chan ast.Directive, 100)
	j.errCh = make((chan error))

	j.wg.Add(1)
	go j.parseRecursively(ctx, j.File)

	// Parse and eventually close input channel
	go func() {
		defer close(j.resCh)
		defer close(j.errCh)
		j.wg.Wait()
	}()
	return j.resCh, j.errCh
}

func (j *RecursiveParser) parseRecursively(ctx context.Context, file string) {
	defer j.wg.Done()
	p, cls, err := FromPath(j.Context, file)
	if err != nil {
		select {
		case j.errCh <- err:
		case <-ctx.Done():
		}
		return
	}
	defer func() {
		err = cls()
		if err != nil {
			select {
			case j.errCh <- err:
			case <-ctx.Done():
			}
		}
	}()

	resCh, errCh := p.ParseAll(ctx)

	for resCh != nil || errCh != nil {
		select {
		case d, ok := <-resCh:
			if !ok {
				resCh = nil
			} else {
				switch t := d.(type) {

				case *ast.Include:
					j.wg.Add(1)
					go j.parseRecursively(ctx, path.Join(filepath.Dir(file), t.Path))

				default:
					select {
					case j.resCh <- d:
					case <-ctx.Done():
						return
					}
				}
			}
		case err, ok := <-errCh:
			if !ok {
				errCh = nil
			} else {
				select {
				case j.errCh <- err:
				case <-ctx.Done():
					return
				}
			}
		}
	}
}
