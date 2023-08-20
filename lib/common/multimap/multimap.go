package multimap

import (
	"github.com/sboehler/knut/lib/common/compare"
	"github.com/sboehler/knut/lib/common/dict"
)

type Node[V any] struct {
	Segment  string
	Value    V
	Children map[string]*Node[V]
	Sorted   []*Node[V]
}

func New[V any](segment string) *Node[V] {
	return &Node[V]{
		Segment:  segment,
		Children: make(map[string]*Node[V]),
	}
}

// GetOrCreate creates or returns the node at the given key.
func (n *Node[V]) GetOrCreate(ss []string) *Node[V] {
	if len(ss) == 0 {
		return n
	}
	head, tail := ss[0], ss[1:]
	return dict.
		GetDefault(n.Children, head, func() *Node[V] { return New[V](head) }).
		GetOrCreate(tail)
}

func (n *Node[V]) Sort(f compare.Compare[*Node[V]]) {
	for _, ch := range n.Children {
		ch.Sort(f)
	}
	n.Sorted = dict.SortedValues(n.Children, f)
}

func SortAlpha[V any](n1, n2 *Node[V]) compare.Order {
	return compare.Ordered(n1.Segment, n2.Segment)
}

func (n *Node[V]) PostOrder(f func(*Node[V])) {
	for _, ch := range n.Children {
		ch.PostOrder(f)
	}
	f(n)
}
