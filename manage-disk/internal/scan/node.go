// Package scan measures where disk space goes under a directory.
package scan

import (
	"sort"
	"strings"
)

// Node is a directory or file in a scan tree.
//
// Sizes are allocated bytes (st_blocks × 512), the number `du` reports, so a
// sparse 60 GB Docker.raw counts the 22 GB it really occupies.
type Node struct {
	Name     string // the root's Name is its absolute path
	IsDir    bool
	Size     int64   // this entry plus, for a directory, everything under it
	Files    int64   // files under a directory; 1 for a file
	Children []*Node // entries at least Options.MinNode big, largest first
	Small    int64   // bytes of the entries too small to keep as nodes
	SmallN   int64   // how many entries were folded into Small
	Skipped  string  // why a directory was not read (see the Skip* constants)
	Parent   *Node

	// skipsInside marks a folder holding a skipped one, so however small it
	// is, it stays in the tree and the "not scanned" note stays reachable.
	skipsInside bool
}

// Path is the node's absolute path.
func (n *Node) Path() string {
	if n.Parent == nil {
		return n.Name
	}
	return n.Parent.Path() + "/" + n.Name
}

// Root walks up to the top of the tree.
func (n *Node) Root() *Node {
	for n.Parent != nil {
		n = n.Parent
	}
	return n
}

// Rel is the node's path relative to the root of its tree ("" for the root).
func (n *Node) Rel() string {
	if n.Parent == nil {
		return ""
	}
	if p := n.Parent.Rel(); p != "" {
		return p + "/" + n.Name
	}
	return n.Name
}

// Find returns the node at absolute path p, or nil when p is outside the tree
// or was folded into a parent's Small total.
func (n *Node) Find(p string) *Node {
	root := n.Root()
	if p == root.Name {
		return root
	}
	rel, ok := strings.CutPrefix(p, root.Name+"/")
	if !ok {
		return nil
	}
	return root.FindRel(rel)
}

// FindRel returns the node at rel, a path relative to n.
func (n *Node) FindRel(rel string) *Node {
	cur := n
	for _, part := range strings.Split(rel, "/") {
		if part == "" {
			continue
		}
		var next *Node
		for _, c := range cur.Children {
			if c.Name == part {
				next = c
				break
			}
		}
		if next == nil {
			return nil
		}
		cur = next
	}
	return cur
}

// Remove drops the node at p from the tree and takes its size off every
// ancestor, so the tree reflects a deletion without a rescan. It reports the
// bytes removed, or false when p is not a node in the tree.
func (n *Node) Remove(p string) (int64, bool) {
	x := n.Find(p)
	if x == nil || x.Parent == nil {
		return 0, false
	}
	parent := x.Parent
	for i, c := range parent.Children {
		if c == x {
			parent.Children = append(parent.Children[:i:i], parent.Children[i+1:]...)
			break
		}
	}
	x.Parent = nil
	adjust(parent, -x.Size, -x.Files)
	return x.Size, true
}

// Replace swaps the node at p for fresh (a rescan of the same path) and
// corrects every ancestor's totals. It reports false when p is not in the tree.
func (n *Node) Replace(p string, fresh *Node) bool {
	x := n.Find(p)
	if x == nil || x.Parent == nil {
		return false
	}
	parent := x.Parent
	fresh.Name = x.Name
	fresh.Parent = parent
	for _, c := range fresh.Children {
		c.Parent = fresh
	}
	for i, c := range parent.Children {
		if c == x {
			parent.Children[i] = fresh
			break
		}
	}
	adjust(parent, fresh.Size-x.Size, fresh.Files-x.Files)
	return true
}

// adjust adds a size and file delta to n and its ancestors, re-sorting each
// level so the largest entries stay first.
func adjust(n *Node, size, files int64) {
	for ; n != nil; n = n.Parent {
		n.Size += size
		n.Files += files
		sortNodes(n.Children)
	}
}

func sortNodes(nodes []*Node) {
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].Size != nodes[j].Size {
			return nodes[i].Size > nodes[j].Size
		}
		return nodes[i].Name < nodes[j].Name
	})
}
