package main

import "net/http"

func main() {
	r := NewEngine()
	r.GET("/", func(c *Context) {
		c.HTML(http.StatusOK, "<h1>Hello World!</h1>")
	})

	r.GET("/hello", func(c *Context) {
		c.String(http.StatusOK, "hello %s, you're at %s\n", c.Query("name"), c.Path)
	})

	r.GET("/hello/:name", nil)

	r.GET("/hello/:u/1", nil)

	r.POST("/login", func(c *Context) {
		c.JSON(http.StatusOK, H{
			"username": c.PostForm("username"),
			"password": c.PostForm("password"),
		})
	})

	r.Run(":9999")
}
