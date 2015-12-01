package webgo

import (
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
)

type Request struct {
	Method    string
	Path      string
	Query     map[string]string
	Headers   http.Header
	Body      []byte
	Arguments []string
}

type Response struct {
	Status     int
	Headers    http.Header
	Body       []byte
	BodyReader io.Reader
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

func Respond(status int, body []byte) *Response {
	return &Response{
		Status:  status,
		Body:    body,
		Headers: make(http.Header),
	}
}

func Redirect(redirectUrl string) *Response {
	resp := Respond(302, []byte{})
	resp.Headers.Set("Location", redirectUrl)
	return resp
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

func (app *Application) Route(pattern string, procFunc ProcessFunc) {
	pattern = strings.TrimRight(pattern, "/")
	if strings.IndexByte(pattern, ' ') < 0 {
		pattern = ".* " + pattern
	}
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
		Process: procFunc,
	})
}

func ParseRequest(r *http.Request) *Request {
	method := r.Method
	path := r.URL.Path
	body, _ := ioutil.ReadAll(r.Body)

	query := make(map[string]string)
	for name, values := range r.URL.Query() {
		query[name] = values[0]
	}

	return &Request{
		Method:  method,
		Path:    path,
		Query:   query,
		Headers: r.Header,
		Body:    body,
	}
}

func ParseResponse(resp *http.Response) *Response {
	status := resp.StatusCode
	body, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	return &Response{
		Status:     status,
		Headers:    resp.Header,
		Body:       body,
		BodyReader: nil,
	}
}

func (app *Application) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	req := ParseRequest(r)
	if req == nil {
		http.Error(w, "Bad Request", 400)
		return
	}

	processor := app.defaultProcessor
	path := strings.TrimRight(req.Path, "/")
	path = req.Method + " " + path
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
	for name, value := range resp.Headers {
		w.Header().Set(name, value[0])
	}
	w.WriteHeader(resp.Status)

	if resp.BodyReader != nil {
		io.Copy(w, resp.BodyReader)
	} else {
		w.Write(resp.Body)
	}
}
