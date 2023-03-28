## Day 1-简单的HTTP处理流程

本次的目标是7天实现一个简单的goWeb框架，这里的所有设计思路都来源于gin，具体实现如下

### 如何处理某个HTTP请求

在之前的网络编程提到过，go中对http请求的处理有两种，第一种是用一个接口去实现`net/server.go`的`Handler`接口从而处理请求，但是这样每次根据业务场景不同都需要定义新的结构体，由此有了第二种处理方式，即定义一个处理函数，我们处理请求的时候只需要按照这个形式实现这个函数即可

```
type HandlerFunc func(http.ResponseWriter, *http.Request)
```

### 用什么数据结构处理HTTP请求

根据gin的设计思想，我们要用一个统一的`Engine`来处理请求，用一个路由映射表`router`去保存K-V对，其中K是请求类型和请求名，如`GET-/getUserById`，V是我们的处理函数`HandlerFunc`，然后就是构造函数，以及添加映射的两个方法`GET`和`POST`

```go
type Engine struct {
	router map[string]HandlerFunc
}

// New is the constructor of gea.Engine
func New() *Engine {
	return &Engine{router: make(map[string]HandlerFunc)}
}

func (engine *Engine) addRoute(method, pattern string, handler HandlerFunc) {
	key := method + "-" + pattern
	engine.router[key] = handler
}

// GET is used to register your GET method to engine router
func (engine *Engine) GET(pattern string, handler HandlerFunc) {
	engine.addRoute("GET", pattern, handler)
}

// POST is used to register your POST method to engine router
func (engine *Engine) POST(pattern string, handler HandlerFunc) {
	engine.addRoute("POST", pattern, handler)
}
```

### 如何全局处理HTTP请求

首先，`Run`函数其实是对`net/http`包下`ListenAndServe`的一个封装，这个方法可以直接启动一个http服务

其次，由于`ListenAndServe`的第二个参数是一个`Handler`类型，也就是说，我们只要传入任意一个实现了这个接口的实例，所有的HTTP请求都会交给这个实例处理，所以`Engine`要去实现这个接口下的`ServeHTTP`方法

```go
// Run is used to start a server by http.ListenAndServe
func (engine *Engine) Run(addr string) (err error) {
	return http.ListenAndServe(addr, engine)
}

func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	key := req.Method + "-" + req.URL.Path
	if handler, ok := engine.router[key]; ok {
		handler(w, req)
	} else {
		fmt.Fprintf(w, "404 NOT FOUND: %s\n", req.URL)
	}
}
```

这样一个最基本的web框架原型就已经搭建完毕了，接下来将会继续完善，例如动态路由，中间件等