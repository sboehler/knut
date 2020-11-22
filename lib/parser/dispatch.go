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

package parser

import (
	"github.com/sboehler/knut/lib/model"
	"io"
	"path"
	"path/filepath"
	"sync"
)

// Parse parses the journal at the path, and branches out for include files
func Parse(path string) (<-chan interface{}, error) {
	ch := make(chan interface{}, 100)

	// Parse and eventually close input channel
	go func() {
		defer close(ch)
		wg := sync.WaitGroup{}
		wg.Add(1)
		parseRecursively(&wg, path, ch)
		wg.Wait()
	}()

	return ch, nil
}

func parseRecursively(wg *sync.WaitGroup, file string, ch chan<- interface{}) {
	defer wg.Done()

	fileCh, err := ParseOneFile(file)
	if err != nil {
		ch <- err
		return
	}
	for r := range fileCh {
		switch d := r.(type) {
		case *model.Include:
			wg.Add(1)
			go parseRecursively(wg, path.Join(filepath.Dir(file), d.Path), ch)
		default:
			ch <- r
		}
	}
}

// ParseOneFile parses one file.
func ParseOneFile(path string) (chan interface{}, error) {
	p, err := Open(path)
	if err != nil {
		return nil, err
	}
	ch := make(chan interface{}, 100)
	go func() {
		defer close(ch)
		for {
			d, err := p.next()
			if err == io.EOF {
				break
			}
			if err != nil {
				ch <- err
			} else {
				ch <- d
			}
		}
	}()
	return ch, nil
}
