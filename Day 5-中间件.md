## Day 5-中间件

### 什么是中间件

中间件也叫middlewares，简单来说就是非业务逻辑的组件，它可以允许用户自己定义一些功能嵌入到框架中，就好像这个框架原生支持一样，所以对于中间件我们需要考虑两个方面：

- 在什么地方插入

  对于使用框架的人来说，他并不关心底层是怎么设计的，如果插入的地方太底层，那么编码就会变得麻烦，如果太接近用户，那么和用户自己显式地调用区别也不大

- 中间件的输入是什么

  中间件的输入决定了其扩展能力，暴露的参数太少，用户能够发挥的空间也有限

### 中间件的设计

中间件的定义与路由映射的 Handler 一致，处理的输入是`Context`对象。插入点是框架接收到请求初始化`Context`对象后，允许用户使用自己定义的中间件做一些额外的处理，例如记录日志等，以及对`Context`进行二次加工。另外通过调用`(*Context).Next()`函数，中间件可等待用户自己定义的 `Handler`处理结束后，再做一些额外的操作，例如计算本次处理所用时间等

中间件是作用于分组的，如果作用于每一个具体的路由的话，和用户显式调用的区别不大

所以我们给每一次请求的Context中加入对应的方法和参数

初始化index为-1，表示当前没有中间件，每调用一次`Next()`方法`index++`，这样就能保证执行所有的中间件且支持嵌套调用

```go
type Context struct {
	// origin object
	Writer http.ResponseWriter
	Req    *http.Request
	// request info
	Path   string
	Method string
	Params map[string]string
	// response info
	StatusCode int
	// middleware
	handlers []HandlerFunc
	index    int
}

func newContext(w http.ResponseWriter, req *http.Request) *Context {
	return &Context{
		Writer: w,
		Req:    req,
		Path:   req.URL.Path,
		Method: req.Method,
		index:  -1,
	}
}

func (c *Context) Next() {
	c.index++
	s := len(c.handlers)
	for ; c.index < s; c.index++ {
		c.handlers[c.index](c)
	}
}
```

考虑以下情况，此时`c.handler`中的顺序依次是`[A, B, Handler]`，此时我们调用Next方法：

1. index++后等于0，小于3，调用`c.handler(0)`即A
2. A执行part1
3. A调用Next()
4. index++后等于1，小于3，调用`c.handler(1)`即B
5. B执行part3
6. B调用Next()
7. index++后等于2，小于3，调用`c.handler(2)`即Handler
8. Handler返回
9. B执行part4
10. B返回
11. A执行part2
12. A返回

从而实现了嵌套调用的过程

```
func A(c *Context) {
	part 1
	c.Next()
	part 2
}

func B(c *Context) {
	part 3
	c.Next()
	part 4
}

func Handler(c *Context) {

}
```

然后就是实现具体的将中间件应用到某个分组的逻辑

`ServeHTTP`的变化在于，当我们接收到一个具体请求时，首先要根据这个请求的路由判断该请求属于哪个分组（即适用于哪些中间件），用`HasPrefix()`实现，得到中间件列表后，赋值给这个请求的handlers `c.handlers`

定义`Use`方法来将某个中间件应用到某个`Group`

handle 函数中，将从路由匹配得到的 Handler 添加到 `c.handlers`列表中，执行`c.Next()`

```go
func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var middlewares []HandlerFunc
	for _, group := range engine.groups {
		if strings.HasPrefix(req.URL.Path, group.prefix) {
			middlewares = append(middlewares, group.middlewares...)
		}
	}
	c := newContext(w, req)
	c.handlers = middlewares
	engine.router.handle(c)
}

func (group *RouterGroup) Use(middlewares ...HandlerFunc) {
	group.middlewares = append(group.middlewares, middlewares...)
}
```

```go
func (r *router) handle(c *Context) {
	n, params := r.getRoute(c.Method, c.Path)
	if n != nil {
		c.Params = params
		key := c.Method + "-" + n.pattern
		c.handlers = append(c.handlers, r.handlers[key])
	} else {
		c.handlers = append(c.handlers, func(c *Context) {
			c.String(http.StatusNotFound, "404 NOT FOUND: %s\n", c.Path)
		})
	}
	c.Next()
}
```

### 测试

```go
func main() {
   r := NewEngine()
   r.Use(Logger()) // global midlleware
   r.GET("/", func(c *Context) {
      c.HTML(http.StatusOK, "<h1>Hello Gee</h1>")
   })

   v2 := r.Group("/v2")
   v2.Use(onlyForV2()) // v2 group middleware
   {
      v2.GET("/hello/:name", func(c *Context) {
         // expect /hello/geektutu
         c.String(http.StatusOK, "hello %s, you're at %s\n", c.Param("name"), c.Path)
      })
   }

   r.Run(":9999")
}
```