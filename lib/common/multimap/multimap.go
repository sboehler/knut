package multimap

import (
	"fmt"

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

// Get creates or returns the node at the given key.
func (n *Node[V]) GetPath(ss []string) (*Node[V], bool) {
	if len(ss) == 0 {
		return n, true
	}
	head, tail := ss[0], ss[1:]
	if child, ok := n.Children[head]; ok {
		return child.GetPath(tail)
	}
	return nil, false
}

// Get gets an immediate child of this node.
func (n *Node[V]) Get(key string) (*Node[V], bool) {
	ch, ok := n.Children[key]
	return ch, ok
}

func (n *Node[V]) MustGet(key string) *Node[V] {
	ch, ok := n.Children[key]
	if !ok {
		panic(fmt.Sprintf("no child with key %s", key))
	}
	return ch
}

// Create creates an immediate child of this node.
func (n *Node[V]) Create(key string) (*Node[V], error) {
	if _, found := n.Children[key]; found {
		return nil, fmt.Errorf("child already exists")
	}
	child := New[V](key)
	n.Children[key] = child
	return child, nil
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
