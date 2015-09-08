# webgo
Simple web framework, Just for fun.

## Quickstart
```go
app := webgo.NewApplication()
app.Route("/fuck", func(req *webgo.Request) *webgo.Response {
  return Respond(200, "fuck you")
})
app.Route(`/user/(\d+)/`, func(req *webgo.Request) *webgo.Response {
  return Respond(200, "user: "+req.Arguments[0])
})
app.Run(":8080")
```
