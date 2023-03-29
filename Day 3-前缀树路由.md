## Day 3-前缀树路由

### 为什么引入前缀树

之前的处理中，我们用一个`map[string]HandlerFunc`来保存路由和其对应的处理方法，这样做的好处当然是对于静态路由，我们可以通过等值查询直接获取到结果，但是如果对于动态路由，即一个Key可能对应多个Value的方式，如果还是用map的话，就会造成大量的哈希冲突，且代码的复杂性也会增加，所以我们引入了一种树状结构前缀树来保存所有的路由，以便解决动态匹配问题

### 怎么实现前缀树

由于http请求中的路径恰好是以`/`为分隔符的，所以前缀树的每一个结点都可以保存一个`/`下的信息，即一个结点的子结点都有相同的前缀，并且这是一个n叉树，这就是http中前缀树的定义，具体实现如下

我们需要考虑的是一个结点中需要保存哪些信息

**需要注意的是，只有当完整路由被注册时，才会把整个pattern赋值给对应的结点，如`/p/:lang/doc`，当我们对这个路由进行匹配时，只有`doc`结点的pattern是`/p/:lang/doc`，而`p`和`:lang`结点的pattern字段都为空**

- pattern：当前结点可以进行匹配的路由，**如果这个路由没有对应的处理方法，值为空**
- part：每一个结点中保存的子路由
- children：子结点的集合
- isWild：整个匹配过程是否是模糊匹配

```go
type node struct {
	pattern  string  // e.g.: /user/:username
	part     string  // e.g.: user
	children *[]node // collection of child nodes
	isWild   bool    // to judge whether match is wild
}
```

接下来要实现的是插入的逻辑和搜索的逻辑，以及对应的结点匹配方法

```go
// matchChild is used to insert a node
func (n *node) matchChild(part string) *node {
	for _, child := range n.children {
		if child.part == part || child.isWild {
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
	child := n.matchChild(part)
	if child == nil {
		child = &node{part: part, isWild: part[0] == ':' || part[0] == '*'}
		n.children = append(n.children, child)
	}
	n.insert(pattern, parts, height+1)
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
```

#### 修改原博的路由冲突问题

原博客中存在一个问题，即如果我们当前有两个路由按照先后顺序`/user/:age`和`/user/18`的话，此时我们去访问`/user/19`这个`pattern`却会调用`/user/18`的逻辑

这是因为我们在插入时没有考虑动态路由和静态路由的冲突问题，根本原因在`insert`方法的递归中止逻辑，它只考虑了新建结点的情况，没考虑已有结点的情况

```go
func (n *node) insert(pattern string, parts []string, height int) {
	if len(parts) == height {
		n.pattern = pattern
		return
	}
    
    // Other Code
}
```

针对这个问题，有两种解决方法：

- 首先是设置优先级，优先静态匹配，其次动态匹配，但是这样也存在问题，如果我们要用URL的形式传参，还是上面那个例子，本来我们要把`18`这个值传给后端，但是现在却变成了访问`/user/18`这个`pattern`

  这不是我们想要的结果，所以对于路由冲突问题，gin的解决方式是把冲突暴露给用户，让用户去调整，尽量避免冲突路由的情况发生

- gin的设计思想是，每一个结点的子结点至多只能有一个动态匹配规则，当我们要插入一个动态匹配时，要对这个结点的子结点进行检查后才能插入，否则直接pannic

以下是摘取出来的一段源码：

`gin/tree.go`

```go
else if n.wildChild {
				// inserting a wildcard node, need to check if it conflicts with the existing wildcard
				n = n.children[len(n.children)-1]
				n.priority++

				// Check if the wildcard matches
				if len(path) >= len(n.path) && n.path == path[:len(n.path)] &&
					// Adding a child to a catchAll is not possible
					n.nType != catchAll &&
					// Check for longer wildcard, e.g. :name and :names
					(len(n.path) >= len(path) || path[len(n.path)] == '/') {
					continue walk
				}

				// Wildcard conflict
				pathSeg := path
				if n.nType != catchAll {
					pathSeg = strings.SplitN(pathSeg, "/", 2)[0]
				}
				prefix := fullPath[:strings.Index(fullPath, pathSeg)] + n.path
				panic("'" + pathSeg +
					"' in new path '" + fullPath +
					"' conflicts with existing wildcard '" + n.path +
					"' in existing prefix '" + prefix +
					"'")
			}
```

所以我们根据这个设计思想，修改如下：

把`match`这个过程拆分成寻找模糊匹配和寻找精确匹配两种，在我们要对一个结点进行插入操作时

- 首先判断当前同级结点是否存在模糊匹配结点，如果存在且插入结点也是模糊匹配就`panic`
- 如果没有模糊匹配结点，那么判断是否有精确匹配，有的话递归调用，没有的话插入，且插入过程都不会受影响

```go
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
```

### Router逻辑修改

这里我们用了一个新的`map[string]*node`来保存两个根结点`GET`和`POST`，其实就是这个map保存了两个前缀树，根结点分别是`GET`和`POST`

```go
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
```

然后就是，当我们得到一个路由的时候，我们要对其进行解析，方便后面参数的传递，所以引入方法`parsePattern`，这里需要说明的是，如果pattern直接以`*`开头，说明任何形式的路由都可以匹配，此时直接跳出循环返回即可

```go
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
```

接下来就是对`addRoute`和`getRoute`的修改和实现

`addRoute`的逻辑是，如果`GET`或`POST`树还没被创建，那么我们就创建对应的根结点，如果已经被创建，那么此时我们要做的就是把我们的路由插入到前缀树中去，此时调用的就是我们刚刚实现的`insert`方法

`getRoute`的逻辑是，root是根结点，我们把需要匹配的路径传入，调用`search`方法，这里的`searchParts`和`parts`的区别是，前者是我们传入的一个静态路由，而后者是我们前缀树中保存的路由，可能是静态的，也可能是动态的，所以我们需要对其进行解析并匹配，`matchResult`就是我们匹配的返回结果

```go
func (r *router) addRoute(method, pattern string, handler HandlerFunc)
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
	params := make(map[string]string)
	root, ok := r.roots[method]

	if !ok {
		return nil, nil
	}

	n := root.search(searchParts, 0)

	if n != nil {
		parts := parsePattern(n.pattern)
		for idx, part := range parts {
			if part[0] == ':' {
				params[part[1:]] = searchParts[idx]
			}
			if part[0] == '*' && len(part) > 1 {
				params[part[1:]] = strings.Join(searchParts[idx:], "/")
                break
			}
		}
		return n, params
	}
	return nil, nil
}
```

### Context和handle修改

对于Context，我们希望有一个结构来保存我们路由匹配的结果，并且`HandlerFunc`能够在调用的过程中解析这个匹配的结果，所以我们把解析的结果存进`Params`，并且可以通过`c.Param("lang")`的方式来获取到对应的值

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
}

func (c *Context) Param(key string) string {
	value, _ := c.Params[key]
	return value
}
```

最后就是修改`router`中`handle`的逻辑

```go
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
```

### 测试

还是启动main函数进行测试

运行

```
curl http://localhost:9999/name/xiaowang
```

得到

```
hello xiaowang!
```

说明我们已经实现了通过URL传参并完成动态匹配的过程了

```
func main() {
	r := NewEngine()
	r.GET("/", func(c *Context) {
		c.HTML(http.StatusOK, "<h1>Hello World!</h1>")
	})

	r.GET("/hello", func(c *Context) {
		c.String(http.StatusOK, "hello %s, you're at %s\n", c.Query("name"), c.Path)
	})

	r.GET("/hello/:name", func(c *Context) {
		c.String(http.StatusOK, "hello %s!\n", c.Param("name"), c.Path)
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

