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

	"github.com/sboehler/knut/lib/common/cpr"
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
func (rp *RecursiveParser) Parse(ctx context.Context) (<-chan ast.Directive, <-chan error) {
	rp.resCh = make(chan ast.Directive, 100)
	rp.errCh = make((chan error))

	rp.wg.Add(1)
	go rp.parseRecursively(ctx, rp.File)

	// Parse and eventually close input channel
	go func() {
		defer close(rp.resCh)
		defer close(rp.errCh)
		rp.wg.Wait()
	}()
	return rp.resCh, rp.errCh
}

func (rp *RecursiveParser) parseRecursively(ctx context.Context, file string) {
	defer rp.wg.Done()
	p, cls, err := FromPath(rp.Context, file)
	if err != nil {
		cpr.Push(ctx, rp.errCh, err)
		return
	}
	defer cls()

	resCh, errCh := p.Parse(ctx)

	for {
		d, ok, err := cpr.Get(resCh, errCh)
		if !ok {
			break
		}
		if err != nil && cpr.Push(ctx, rp.errCh, err) != nil {
			return
		}
		if t, ok := d.(*ast.Include); ok {
			rp.wg.Add(1)
			go rp.parseRecursively(ctx, path.Join(filepath.Dir(file), t.Path))
			continue
		}
		if cpr.Push(ctx, rp.resCh, d) != nil {
			return
		}
	}
}
