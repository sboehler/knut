package journal

import (
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

// Parse parses the journal at the path, and branches out for include files
func (j *Journal) Parse() chan interface{} {
	var (
		ch = make(chan interface{}, 100)
		wg sync.WaitGroup
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
	for d := range p.ParseAll() {
		switch t := d.(type) {
		case error:
			return t

		case *ledger.Include:
			wg.Add(1)
			go func() {
				if err := j.parseRecursively(wg, ch, path.Join(filepath.Dir(file), t.Path)); err != nil {
					ch <- err
				}
				wg.Done()
			}()
		default:
			ch <- d
		}
	}
	return nil
}
