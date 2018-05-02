// Package domaintree8 implements radix tree data structure for domain names.
package domaintree8

// !!!DON'T EDIT!!! Generated with infobloxopen/go-trees/etc from domaintree{{.bits}} with etc -s uint8 -d dtuintX.yaml -t ./domaintree\{\{.bits\}\}

import (
	"github.com/infobloxopen/go-trees/domain"
)

// Node is a radix tree for domain names.
type Node struct {
	branches *labelTree

	hasValue bool
	value    uint8
}

// Pair represents a key-value pair returned by Enumerate method.
type Pair struct {
	Key   string
	Value uint8
}

// Insert puts value using given domain as a key. The method returns new tree (old one remains unaffected). Input name converted to ASCII lowercase according to RFC-4343 (by mapping A-Z to a-z) to perform case-insensitive comparison when getting data from the tree.
func (n *Node) Insert(d string, v uint8) *Node {
	if n == nil {
		n = &Node{}
	} else {
		n = &Node{
			branches: n.branches,
			hasValue: n.hasValue,
			value:    n.value}
	}
	r := n

	for _, label := range domain.Split(d) {
		next, ok := n.branches.rawGet(label)
		if ok {
			next = &Node{
				branches: next.branches,
				hasValue: next.hasValue,
				value:    next.value}
		} else {
			next = &Node{}
		}

		n.branches = n.branches.rawInsert(label, next)
		n = next
	}

	n.hasValue = true
	n.value = v

	return r
}

// InplaceInsert puts or replaces value using given domain as a key. The method inserts data directly to current tree so make sure you have exclusive access to it. Input name converted in the same way as for Insert.
func (n *Node) InplaceInsert(d string, v uint8) {
	if n.branches == nil {
		n.branches = newLabelTree()
	}

	for _, label := range domain.Split(d) {
		next, ok := n.branches.rawGet(label)
		if ok {
			n = next
		} else {
			next := &Node{branches: newLabelTree()}
			n.branches.rawInplaceInsert(label, next)
			n = next
		}
	}

	n.hasValue = true
	n.value = v
}

// Enumerate returns key-value pairs in given tree sorted by key first by top level domain label second by second level and so on.
func (n *Node) Enumerate() chan Pair {
	ch := make(chan Pair)

	go func() {
		defer close(ch)
		n.enumerate("", ch)
	}()

	return ch
}

// Get gets value for domain which is equal to domain in the tree or is a subdomain of existing domain.
func (n *Node) Get(d string) (uint8, bool) {
	if n == nil {
		return 0, false
	}

	for _, label := range domain.Split(d) {
		next, ok := n.branches.rawGet(label)
		if !ok {
			break
		}

		n = next
	}

	return n.value, n.hasValue
}

// WireGet gets value for domain which is equal to domain in the tree or is a subdomain of existing domain. The method accepts domain name in "wire" format described by RFC-1035 section "3.1. Name space definitions". Additionally it requires all ASCII letters (A-Z) to be converted to their lowercase counterparts (a-z). Returns error in case of compressed names (label length > 63 octets), malformed domain names (last label length too big) and too long domain names (more than 255 bytes).
func (n *Node) WireGet(d domain.WireNameLower) (uint8, bool, error) {
	if n == nil {
		return 0, false, nil
	}

	err := domain.WireSplitCallback(d, func(label []byte) bool {
		if next, ok := n.branches.rawGet(label); ok {
			n = next
			return true
		}

		return false
	})
	if err != nil {
		return 0, false, err
	}

	return n.value, n.hasValue, nil
}

// DeleteSubdomains removes current domain and all its subdomains if any. It returns new tree and flag if deletion indeed occurs.
func (n *Node) DeleteSubdomains(d string) (*Node, bool) {
	if n == nil {
		return nil, false
	}

	labels := domain.Split(d)
	if len(labels) > 0 {
		return n.delSubdomains(domain.Split(d))
	}

	if n.hasValue || !n.branches.isEmpty() {
		return &Node{}, true
	}

	return n, false
}

// Delete removes current domain only. It returns new tree and flag if deletion indeed occurs.
func (n *Node) Delete(d string) (*Node, bool) {
	if n == nil {
		return nil, false
	}

	labels := domain.Split(d)
	if len(labels) > 0 {
		return n.del(domain.Split(d))
	}

	if n.hasValue {
		if n.branches.isEmpty() {
			return &Node{}, true
		}

		return &Node{branches: n.branches}, true
	}

	return n, false
}

func (n *Node) enumerate(s string, ch chan Pair) {
	if n == nil {
		return
	}

	if n.hasValue {
		ch <- Pair{
			Key:   s,
			Value: n.value}
	}

	for item := range n.branches.rawEnumerate() {
		sub := item.Key.String()
		if len(s) > 0 {
			sub += "." + s
		}

		item.Value.enumerate(sub, ch)
	}
}

func (n *Node) delSubdomains(labels []domain.Label) (*Node, bool) {
	label := labels[0]
	if len(labels) > 1 {
		next, ok := n.branches.rawGet(label)
		if !ok {
			return n, false
		}

		next, ok = next.delSubdomains(labels[1:])
		if !ok {
			return n, false
		}

		if next.branches.isEmpty() && !next.hasValue {
			branches, _ := n.branches.rawDel(label)
			return &Node{
				branches: branches,
				hasValue: n.hasValue,
				value:    n.value}, true
		}

		return &Node{
			branches: n.branches.rawInsert(label, next),
			hasValue: n.hasValue,
			value:    n.value}, true
	}

	branches, ok := n.branches.rawDel(label)
	if ok {
		return &Node{
			branches: branches,
			hasValue: n.hasValue,
			value:    n.value}, true
	}

	return n, false
}

func (n *Node) del(labels []domain.Label) (*Node, bool) {
	label := labels[0]
	next, ok := n.branches.rawGet(label)
	if !ok {
		return n, false
	}
	if len(labels) > 1 {
		next, ok = next.del(labels[1:])
		if !ok {
			return n, false
		}

		if next.branches.isEmpty() && !next.hasValue {
			branches, _ := n.branches.rawDel(label)
			return &Node{
				branches: branches,
				hasValue: n.hasValue,
				value:    n.value}, true
		}

		return &Node{
			branches: n.branches.rawInsert(label, next),
			hasValue: n.hasValue,
			value:    n.value}, true
	}

	if next.branches.isEmpty() {
		branches, _ := n.branches.rawDel(label)
		return &Node{
			branches: branches,
			hasValue: n.hasValue,
			value:    n.value}, true
	}

	return &Node{
		branches: n.branches.rawInsert(label, &Node{branches: next.branches}),
		hasValue: n.hasValue,
		value:    n.value}, true
}
