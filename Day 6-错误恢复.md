## Day 6-错误恢复

如果因为用户的非法操作，如空指针，数组越界等行为导致整个web服务宕机，这显然不是我们想要的结果，所以我们需要一定的逻辑，使得用户即使发生了非法操作，也不会导致整个web服务宕机，而是向用户返回`Internal Error`，后端定位到发生错误的位置

### 如何实现错误恢复

之前我们实现了中间件，可以借由中间件的方式来实现错误恢复

`gee/recovery.go`

这个逻辑很简单，由于defer会在错误发生之后执行完再退出，`recover()`会尝试恢复已经发生的`panic()`（类似java中的try发生异常被catch捕获，之后控制权交给catch），同理，recover后控制权交给recover，逻辑处理完后再回到主函数调用`c.Next()`

那么我们在defer上挂载这个错误恢复函数，然后把堆栈信息打印在日志中，并且向用户返回`500`

`trace`函数是用于打印堆栈信息的，具体过程是如下

- `runtime.Callers()`是用来返回调用栈的程序计数器，0是Caller本身，1是上一层trace，2是再上一层的defer func，所以我们跳过这三个Caller
- 接下来，通过 `runtime.FuncForPC(pc)` 获取对应的函数，在通过 `fn.FileLine(pc)` 获取到调用该函数的文件名和行号，打印在日志中。

```go
func Recovery() HandlerFunc {
	return func(c *Context) {
		defer func() {
			if err := recover(); err != nil {
				message := fmt.Sprintf("%s", err)
				log.Printf("%s\n\n", trace(message))
				c.Fail(http.StatusInternalServerError, "Internal Server Error")
			}
		}()

		c.Next()
	}
}

func trace(message string) string {
	var pcs [32]uintptr
	n := runtime.Callers(3, pcs[:]) // skip first 3 caller

	var str strings.Builder
	str.WriteString(message + "\nTraceback:")
	for _, pc := range pcs[:n] {
		fn := runtime.FuncForPC(pc)
		file, line := fn.FileLine(pc)
		str.WriteString(fmt.Sprintf("\n\t%s:%d", file, line))
	}
	return str.String()
}
```

### 测试

```go
func main() {
	r := Default()
	r.GET("/", func(c *Context) {
		c.String(http.StatusOK, "Hello Geektutu\n")
	})
	// index out of range for testing Recovery()
	r.GET("/panic", func(c *Context) {
		names := []string{"geektutu"}
		c.String(http.StatusOK, names[100])
	})

	r.Run(":9999")
}
```

