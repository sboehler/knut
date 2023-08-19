package multimap

import (
	"github.com/sboehler/knut/lib/common/compare"
	"github.com/sboehler/knut/lib/common/dict"
)

type Node[V any] struct {
	Segment string
	Value   V
	Weight  float64

	children map[string]*Node[V]
}

func New[V any](segment string) *Node[V] {
	return &Node[V]{
		Segment:  segment,
		children: make(map[string]*Node[V]),
	}
}

// GetOrCreate creates or returns the node at the given key.
func (n *Node[V]) GetOrCreate(ss []string) *Node[V] {
	if len(ss) == 0 {
		return n
	}
	head, tail := ss[0], ss[1:]
	return dict.
		GetDefault(n.children, head, func() *Node[V] {
			return New[V](head)
		}).
		GetOrCreate(tail)
}

func (n *Node[V]) Children() []*Node[V] {
	return dict.Values[string, *Node[V]](n.children)
}

func (n *Node[V]) SortedChildren() []*Node[V] {
	return dict.SortedValues(n.children, Compare)
}

func Compare[V any](n1, n2 *Node[V]) compare.Order {
	if n1.Weight != n2.Weight {
		return compare.Ordered(n1.Weight, n2.Weight)
	}
	return compare.Ordered(n1.Segment, n2.Segment)
}

func (n *Node[V]) SortedChildrenFunc(f compare.Compare[*Node[V]]) []*Node[V] {
	return dict.SortedValues(n.children, f)
}

func (n *Node[V]) PostOrder(f func(*Node[V])) {
	for _, ch := range n.children {
		ch.PostOrder(f)
	}
	f(n)
}
