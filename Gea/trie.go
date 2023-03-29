package main

import (
	"strings"
)

type node struct {
	pattern  string  // e.g.: /user/:username
	part     string  // e.g.: :username
	children []*node // collection of child nodes
	isWild   bool    // to judge whether match is wild
}

// matchWildChild is used to find a node which is wild
func (n *node) matchWildChild() *node {
	for _, child := range n.children {
		if child.isWild {
			return child
		}
	}
	return nil
}

// matchAccurateChild is used to find an accurate node which node.part == part
func (n *node) matchAccurateChild(part string) *node {
	for _, child := range n.children {
		if child.part == part {
			return child
		}
	}
	return nil
}

// matchChildren is used to search a set of node
func (n *node) matchChildren(part string) []*node {
	nodes := make([]*node, 0)
	for _, child := range n.children {
		if child.part == part || child.isWild {
			nodes = append(nodes, child)
		}
	}
	return nodes
}

func (n *node) insert(pattern string, parts []string, height int) {
	if len(parts) == height {
		n.pattern = pattern
		return
	}

	part := parts[height]
	// 先找模糊匹配
	child := n.matchWildChild()
	if child != nil && (part[0] == ':' || part[0] == '*') {
		panic(part + " in new path " + pattern + " conflicts with existing wildcard " + child.part)
	}
	// 再找静态匹配
	child = n.matchAccurateChild(part)
	if child == nil {
		child = &node{part: part, isWild: part[0] == ':' || part[0] == '*'}
		n.children = append(n.children, child)
	}
	child.insert(pattern, parts, height+1)
}

func (n *node) search(parts []string, height int) *node {
	if len(parts) == height || strings.HasPrefix(n.part, "*") {
		if n.pattern == "" {
			return nil
		}
		return n
	}

	part := parts[height]
	children := n.matchChildren(part)

	for _, child := range children {
		result := child.search(parts, height+1)
		if result != nil {
			return result
		}
	}
	return nil
}

// traverse is used to collect all nodes which node.pattern is not null
func (n *node) traverse(list *[]*node) {
	if n.pattern != "" {
		*list = append(*list, n)
	}
	for _, child := range n.children {
		child.traverse(list)
	}
}
