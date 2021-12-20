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

package past

import (
	"go.uber.org/multierr"
)

// Initializer gets called before processing.
type Initializer interface {
	Initialize(l *PAST) error
}

// Processor processes the balance and the ledger day.
type Processor interface {
	Process(d *Day) error
}

// Finalizer gets called after all days have been processed.
type Finalizer interface {
	Finalize() error
}

// Sync processes an AST.
func Sync(l *PAST, steps []Processor) (resErr error) {
	for _, pr := range steps {
		if f, ok := pr.(Initializer); ok {
			if err := f.Initialize(l); err != nil {
				return err
			}
		}
	}
	defer func() {
		for _, pr := range steps {
			if f, ok := pr.(Finalizer); ok {
				if err := f.Finalize(); err != nil {
					resErr = multierr.Append(resErr, err)
				}
			}
		}
	}()
	for _, day := range l.Days {
		for _, pr := range steps {
			if err := pr.Process(day); err != nil {
				return err
			}
		}
	}
	return nil
}

// Async processes the AST asynchronously.
func Async(l *PAST, steps []Processor) <-chan error {
	errCh := make(chan error)
	go func(steps []Processor) {
		defer close(errCh)
		if err := Sync(l, steps); err != nil {
			errCh <- err
			return
		}
	}(steps)
	return errCh
}
