package slice

import (
	"testing"
)

type input struct {
	a, b, c int
}

func TestParallel(t *testing.T) {
	const size = 100000
	var list []*input
	for i := 0; i < size; i++ {
		list = append(list, &input{i, i + 1, i + 2})
	}
	fnA := func(in *input) error {
		in.a++
		return nil
	}
	fnB := func(in *input) error {
		in.b = in.a + in.b
		return nil
	}
	fnC := func(in *input) error {
		in.c = in.c + in.b
		return nil
	}
	got, err := Parallel(list, fnA, fnB, fnC)
	if err != nil {
		t.Fatalf("Parallel() returned unexpected error: %v", err)
	}
	for i, l := range got {
		if l.a != i+1 || l.b != 2*(i+1) || l.c != 3*i+4 {
			t.Fatalf("invalid test[%d]: %v", i, l)
		}
	}
}
