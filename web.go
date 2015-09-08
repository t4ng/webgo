package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
)

type Request struct {
	Method    string
	Path      string
	Params    map[string]interface{}
	Headers   map[string]string
	Body      []byte
	Context   interface{}
	Arguments []string
}

type Response struct {
	Status  int
	Headers map[string]string
	Body    []byte
}

type Processor struct {
	Match   func(path string) (bool, []string)
	Process func(*Request) *Response
}

type Application struct {
	httpServer       *http.Server
	processors       []*Processor
	defaultProcessor *Processor
}

func NewApplication() *Application {
	server := &http.Server{}
	app := &Application{
		httpServer: server,
	}
	server.Handler = app
	return app
}

func (app *Application) Run(addr string) {
	app.httpServer.Addr = addr
	app.httpServer.ListenAndServe()
}

func (app *Application) SetDefaultProcessor(p *Processor) {
	app.defaultProcessor = p
}

func (app *Application) AddProcessor(p *Processor) {
	app.processors = append(app.processors, p)
}

type ProcessFunc func(*Request) *Response

func (app *Application) Route(pattern string, fn ProcessFunc) {
	pattern = strings.TrimRight(pattern, "/")
	pattern = "^" + pattern + "$"
	re, _ := regexp.Compile(pattern)

	matchFunc := func(path string) (bool, []string) {
		if re.MatchString(path) {
			return true, re.FindStringSubmatch(path)[1:]
		}
		return false, []string{}
	}

	app.AddProcessor(&Processor{
		Match:   matchFunc,
		Process: fn,
	})
}

func (app *Application) ParseRequest(r *http.Request) *Request {
	method := r.Method
	path := r.URL.Path
	body := []byte{}
	params := map[string]interface{}{}

	if method == "GET" {
		query := r.URL.Query()
		params = make(map[string]interface{})
		for name, values := range query {
			params[name] = values[0]
		}
	} else {
		body, _ = ioutil.ReadAll(r.Body)
		var iface interface{}
		err := json.Unmarshal(body, &iface)
		if err != nil {
			return nil
		}
		params = iface.(map[string]interface{})
	}

	return &Request{
		Method:  method,
		Path:    path,
		Params:  params,
		Body:    body,
		Context: app,
	}
}

func (app *Application) ParseResponse(resp *http.Response) *Response {
	status := resp.StatusCode
	headers := make(map[string]string)
	body, _ := ioutil.ReadAll(resp.Body)

	for name, _ := range resp.Header {
		headers[name] = resp.Header.Get(name)
	}

	return &Response{
		Status:  status,
		Headers: headers,
		Body:    body,
	}
}

func (app *Application) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	req := app.ParseRequest(r)
	if req == nil {
		http.Error(w, "Bad Request", 400)
		return
	}

	processor := app.defaultProcessor
	path := strings.TrimRight(req.Path, "/")
	for _, p := range app.processors {
		if ok, args := p.Match(path); ok {
			processor = p
			req.Arguments = args
			break
		}
	}

	if processor == nil {
		http.Error(w, "Not Found", 404)
		return
	}

	resp := processor.Process(req)
	w.WriteHeader(resp.Status)
	for name, value := range resp.Headers {
		w.Header().Set(name, value)
	}
	w.Write(resp.Body)
}
