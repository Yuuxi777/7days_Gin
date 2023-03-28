## Day 2-上下文

今天的主要目标是

- 抽象出结构体`Context`对http`Request`和`Response`进行封装，提供JSON、HTML、文本等返回形式
- 将`Engine`中的`router`部分独立处理，方便后续开发

### 为什么需要上下文

这里的Context和并发编程中不是一个概念，这里的上下文其实完全可以理解成一次http请求，它包括了处理http请求的`http.ResponseWriter`以及`*http.Request`等等，Context的生命周期和一次http的请求一样，它在一个请求发起时被建立，在一个请求返回后被销毁，这一次请求的所有信息都会被保存在这个结构当中，从而简化了接口

#### 复用http请求的相关字段

假设此刻我们是一个使用框架的用户，如果我们在发起请求的时候，除了我们的data，还要对http请求的其他字段如`Content-Type` \ `Header`等进行显式地设置，那么将会导致编码的效率极大地降低，且对用户不友好，所以我们要封装http请求的上下文，从而使得用户在使用时只需要关心自己需要请求的数据，而对其他的字段不敏感

下面是一个封装前与封装后的代码实现对比：

封装前

```go
obj = map[string]interface{}{
    "name": "geektutu",
    "password": "1234",
}
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusOK)
encoder := json.NewEncoder(w)
if err := encoder.Encode(obj); err != nil {
    http.Error(w, err.Error(), 500)
}
```

封装后：

```go
c.JSON(http.StatusOK, gee.H{
    "username": c.PostForm("username"),
    "password": c.PostForm("password"),
})
```

#### 保存一次http请求的所有信息

对于一个框架来说，我们可能还有别的功能，例如解析动态路由，那么动态路由的值需要一个结构来进行保存，对于框架使用的中间件也是同理，用一个结构Context，我们可以轻易地保存一次请求中所有我们需要的信息

### 具体实现

`Gea/context.go`

首先我们设置了一个新的类型`H`以方便在后续以json形式返回结果的时候能够快速编码

接下来是Context结构体以及它的构造函数，可以看到它包含了http请求的一些字段，包括请求的路径，请求的方法，请求的返回状态码等等

```go
type H map[string]interface{}

type Context struct {
	Writer     http.ResponseWriter // origin object
	Req        *http.Request       // origin object
	Path       string              // request info
	Method     string              // request info
	StatusCode int                 // response info
}

var (
	contentType = "Content-Type"
	jsonType    = "application/json"
	plainType   = "text/plain"
	htmlType    = "text/html"
)

func newContext(w http.ResponseWriter, req *http.Request) *Context {
	return &Context{
		Writer: w,
		Req:    req,
		Path:   req.URL.Path,
		Method: req.Method,
	}
}
```

- `PostForm`和`Query`方法是用来接收两种类型的参数的，即`Form`和`Query`形式
- `JSON`  / `HTML` / `Data` / `HTML` 是用于快速构建对应类型的返回值，而无需再去显式声明`Content-Type`字段

```go
func (c *Context) PostForm(key string) string {
	return c.Req.FormValue(key)
}

func (c *Context) Query(key string) string {
	return c.Req.URL.Query().Get(key)
}

func (c *Context) Status(code int) {
	c.StatusCode = code
	c.Writer.WriteHeader(code)
}

func (c *Context) SetHeader(key, value string) {
	c.Writer.Header().Set(key, value)
}

func (c *Context) String(code int, format string, values ...interface{}) {
	c.SetHeader(contentType, plainType)
	c.Status(code)
	_, err := c.Writer.Write([]byte(fmt.Sprintf(format, values...)))
	if err != nil {
		return
	}
}

func (c *Context) JSON(code int, object interface{}) {
	c.SetHeader(contentType, jsonType)
	c.Status(code)
	encoder := json.NewEncoder(c.Writer)
	if err := encoder.Encode(object); err != nil {
		http.Error(c.Writer, err.Error(), 500)
	}
}

func (c *Context) Data(code int, data []byte) {
	c.Status(code)
	_, err := c.Writer.Write(data)
	if err != nil {
		return
	}
}

func (c *Context) HTML(code int, html string) {
	c.SetHeader(contentType, htmlType)
	c.Status(code)
	_, err := c.Writer.Write([]byte(html))
	if err != nil {
		return
	}
}
```



`Gea/router.go`

我们把路由部分抽离出来，重新构建一个新的结构体

`addRoute`方法和`handle`方法没有过多区别，一个把请求注册到map中去，一个从map中取出请求对应的方法并执行，唯一有区别的就是`handle`的入参由`http.ResponseWriter, *http.Request`变成了`*Context`，因为我们已经对这两个字段做了封装处理了

```go
type router struct {
	handlers map[string]HandlerFunc
}

func newRouter() *router {
	return &router{handlers: make(map[string]HandlerFunc)}
}

func (r *router) addRoute(method, pattern string, handler HandlerFunc) {
	log.Printf("Route %4s - %s", method, pattern)
	key := method + "-" + pattern
	r.handlers[key] = handler
}

func (r *router) handle(c *Context) {
	key := c.Method + "-" + c.Path
	if handler, ok := r.handlers[key]; ok {
		handler(c)
	} else {
		c.String(http.StatusNotFound, "404 NOT FOUND: %s\n", c.Path)
	}
}
```



`Gea/gea.go`

最后就是对框架的入口进行修改，把原来的map换成`*router`，且`HandlerFunc`的入参也由那两个字段变成`*Context`

```go
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
```

### 测试

最后在main函数中启动服务测试

```go
func main() {
	r := NewEngine()
	r.GET("/", func(c *Context) {
		c.HTML(http.StatusOK, "<h1>Hello World!</h1>")
	})

	r.GET("/hello", func(c *Context) {
		c.String(http.StatusOK, "hello %s, you're at %s\n", c.Query("name"), c.Path)
	})

	r.POST("/login", func(c *Context) {
		c.JSON(http.StatusOK, H{
			"username": c.PostForm("username"),
			"password": c.PostForm("password"),
		})
	})

	r.Run(":9999")
}
```