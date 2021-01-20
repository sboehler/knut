package journal

import (
	"io"
	"path"
	"path/filepath"
	"sync"

	"github.com/sboehler/knut/lib/ledger"
	"github.com/sboehler/knut/lib/parser"
)

// Journal represents a journal on disk.
type Journal struct {
	File string
}

// ToLedger processes a Journal and returns a ledger.
func (j *Journal) ToLedger(f ledger.Filter) (ledger.Ledger, error) {
	l, err := ledger.Build(f, j.Parse())
	if err != nil {
		return nil, err
	}
	return l, nil
}

// Parse parses the journal at the path, and branches out for include files
func (j *Journal) Parse() chan interface{} {
	var (
		ch = make(chan interface{}, 100)
		wg = sync.WaitGroup{}
	)

	// Parse and eventually close input channel
	go func() {
		defer close(ch)
		wg.Add(1)
		go func() {
			if err := j.parseRecursively(&wg, ch, j.File); err != nil {
				ch <- err
			}
			wg.Done()
		}()
		wg.Wait()
	}()
	return ch
}

func (j *Journal) parseRecursively(wg *sync.WaitGroup, ch chan<- interface{}, file string) error {
	p, err := parser.Open(file)
	if err != nil {
		return err
	}
	for {
		d, err := p.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if i, ok := d.(*ledger.Include); ok {
			wg.Add(1)
			go func() {
				if err := j.parseRecursively(wg, ch, path.Join(filepath.Dir(file), i.Path)); err != nil {
					ch <- err
				}
				wg.Done()
			}()
		} else {
			ch <- d
		}
	}
}
