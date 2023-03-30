## Day 4-分组控制

分组，指的是对路由进行分组，即对于不同的路由，我们应该有不同的逻辑去处理，例如`/post`开头的路由可以匿名访问，`/admin`开头的路由需要鉴权等，我们通过在不同分组上实现不同的中间件，从而实现对不同分组逻辑处理的不同，大致的代码如下：

根据这个代码我们可以看出来，v1这个`Group`对象需要有注册路由的能力，即我们传入一个`/v1`，它可以去调用`addRoute()`方法，此外，它还要有访问`GET`方法的能力

```go
r := gee.New()
v1 := r.Group("/v1")
v1.GET("/", func(c *gee.Context) {
	c.HTML(http.StatusOK, "<h1>Hello Gee</h1>")
})
```

所以我们这样进行拆分

相当于，我们把路由部分的所有功能都交给`RouterGroup`去实现，包括`GET`、`POST`、`addRoute`等，此时Engine里有一个`RouterGroup`的指针，相当于Engine去“继承”了这个字段，能够访问这个字段的一系列方法，而Engine只需要去负责整个框架的启动和处理即可

```go
type Engine struct {
	*RouterGroup
	router *router
	groups []*RouterGroup
}

type RouterGroup struct {
	prefix      string
	middlewares []HandlerFunc
	parent      *RouterGroup
	engine      *Engine
}
```

最后就是对几个方法进行简单修改

```go
// NewEngine is the constructor of gea.Engine
func NewEngine() *Engine {
	engine := &Engine{router: newRouter()}
	engine.RouterGroup = &RouterGroup{engine: engine}
	engine.groups = []*RouterGroup{engine.RouterGroup}
	return engine
}

// Group is defined to create a new RouterGroup
// remember all groups share the same Engine instance
func (group *RouterGroup) Group(prefix string) *RouterGroup {
	engine := group.engine
	newGroup := &RouterGroup{
		prefix: group.prefix + prefix,
		engine: engine,
	}
	engine.groups = append(engine.groups, newGroup)
	return newGroup
}

func (group *RouterGroup) addRoute(method string, comp string, handler HandlerFunc) {
	pattern := group.prefix + comp
	log.Printf("Route %4s - %s", method, pattern)
	group.engine.router.addRoute(method, pattern, handler)
}

// GET is used to register your GET method to engine router
func (group *RouterGroup) GET(pattern string, handler HandlerFunc) {
	group.addRoute("GET", pattern, handler)
}

// POST is used to register your POST method to engine router
func (group *RouterGroup) POST(pattern string, handler HandlerFunc) {
	group.addRoute("POST", pattern, handler)
}

// Run is used to start a server by http.ListenAndServe
func (engine *Engine) Run(addr string) (err error) {
	return http.ListenAndServe(addr, engine)
}

func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	c := newContext(w, req)
	engine.router.handle(c)
}

```

