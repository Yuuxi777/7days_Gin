package main

import (
	"net/http"
)

type HandlerFunc func(c *Context)

type Engine struct {
	router *router
}

// NewEngine is the constructor of gea.Engine
func NewEngine() *Engine {
	return &Engine{router: newRouter()}
}

// GET is used to register your GET method to engine router
func (engine *Engine) GET(pattern string, handler HandlerFunc) {
	engine.router.addRoute("GET", pattern, handler)
}

// POST is used to register your POST method to engine router
func (engine *Engine) POST(pattern string, handler HandlerFunc) {
	engine.router.addRoute("POST", pattern, handler)
}

// Run is used to start a server by http.ListenAndServe
func (engine *Engine) Run(addr string) (err error) {
	return http.ListenAndServe(addr, engine)
}

func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	c := newContext(w, req)
	engine.router.handle(c)
}
