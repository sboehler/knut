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

package ledger

// Initializer gets called before processing.
type Initializer interface {
	Initialize(l Ledger) error
}

// Process processes the balance and the ledger day.
type Process interface {
	Process(d *Day) error
}

// Finalizer gets called after all days have been processed.
type Finalizer interface {
	Finalize() error
}

// Processor processes a ledger.
type Processor struct {
	Steps []Process
}

// Process processes a ledger.
func (b Processor) Process(l Ledger) error {
	for _, pr := range b.Steps {
		if f, ok := pr.(Initializer); ok {
			if err := f.Initialize(l); err != nil {
				return err
			}
		}
	}
	for _, day := range l.Days {
		for _, pr := range b.Steps {
			if err := pr.Process(day); err != nil {
				return err
			}
		}
	}
	for _, pr := range b.Steps {
		if f, ok := pr.(Finalizer); ok {
			if err := f.Finalize(); err != nil {
				return err
			}
		}
	}
	return nil
}
