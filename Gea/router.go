package main

import (
	"net/http"
	"strings"
)

type router struct {
	roots    map[string]*node // roots is only used to store root node "GET" and "POST"
	handlers map[string]HandlerFunc
}

func newRouter() *router {
	return &router{
		roots:    make(map[string]*node),
		handlers: make(map[string]HandlerFunc),
	}
}

func parsePattern(pattern string) []string {
	parts := make([]string, 0)

	for _, subPattern := range strings.Split(pattern, "/") {
		if subPattern != "" {
			parts = append(parts, subPattern)
			if subPattern[0] == '*' {
				break
			}
		}
	}
	return parts
}

func (r *router) addRoute(method, pattern string, handler HandlerFunc) {
	parts := parsePattern(pattern)

	// `method` field has only two values: "GET", "POST"
	key := method + "-" + pattern
	if _, ok := r.roots[method]; !ok {
		r.roots[method] = &node{}
	}
	r.roots[method].insert(pattern, parts, 0)
	r.handlers[key] = handler
}

func (r *router) getRoute(method, path string) (*node, map[string]string) {
	searchParts := parsePattern(path)
	matchResult := make(map[string]string)
	root, ok := r.roots[method]

	if !ok {
		return nil, nil
	}

	n := root.search(searchParts, 0)

	if n != nil {
		parts := parsePattern(n.pattern)
		for idx, part := range parts {
			if part[0] == ':' {
				matchResult[part[1:]] = searchParts[idx]
			}
			if part[0] == '*' && len(part) > 1 {
				matchResult[part[1:]] = strings.Join(searchParts[idx:], "/")
				break
			}
		}
		return n, matchResult
	}
	return nil, nil
}

func (r *router) handle(c *Context) {
	n, params := r.getRoute(c.Method, c.Path)
	if n != nil {
		c.Params = params
		key := c.Method + "-" + n.pattern
		r.handlers[key](c)
	} else {
		c.String(http.StatusNotFound, "404 NOT FOUND: %s\n", c.Path)
	}
}

func (r *router) getRoutes(method string) []*node {
	root, ok := r.roots[method]
	if !ok {
		return nil
	}
	nodes := make([]*node, 0)
	root.traverse(&nodes)
	return nodes
}
