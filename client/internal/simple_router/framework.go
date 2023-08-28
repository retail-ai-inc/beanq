package simple_router

import (
	"context"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/retail-ai-inc/beanq/helper/json"
)

type (
	Routes struct {
		Path      string
		Method    string
		HandleFun HandlerFunc
	}
	Context struct {
		*http.ServeMux
		routes   []*Routes
		request  *http.Request
		response http.ResponseWriter
	}

	HandlerFunc func(ctx *Context) error
)

var beanqCtx = sync.Pool{New: func() any {
	return &Context{}
}}

func New() *Context {
	return &Context{
		ServeMux: http.NewServeMux(),
		routes:   make([]*Routes, 0),
	}
}

func (t *Context) StaticFile(fileName string) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		url := r.RequestURI
		if strings.HasSuffix(url, ".vue") {
			w.Header().Set("Content-Type", "application/octet-stream")
		}
		var dir string = "./"
		_, f, _, ok := runtime.Caller(0)
		if ok {
			dir = filepath.Dir(f)
		}

		hdl := http.FileServer(http.Dir(path.Join(dir, fileName)))
		hdl.ServeHTTP(w, r)
		return
	}

}

func (t *Context) add(path, method string, handleFunc HandlerFunc) {
	t.routes = append(t.routes, &Routes{
		Path:      path,
		Method:    method,
		HandleFun: handleFunc,
	})
}

func (t *Context) Get(path string, handleFunc HandlerFunc) {
	t.add(path, "GET", handleFunc)
}

func (t *Context) Post(path string, handleFunc HandlerFunc) {
	t.add(path, "POST", handleFunc)
}

func (t *Context) Delete(path string, handleFunc HandlerFunc) {
	t.add(path, "DELETE", handleFunc)
}

func (t *Context) Put(path string, handleFunc HandlerFunc) {
	t.add(path, "PUT", handleFunc)
}

func (t *Context) Head(path string, handlerFunc HandlerFunc)   {}
func (t *Context) Patch(path string, handlerFunc HandlerFunc)  {}
func (t *Context) Option(path string, handlerFunc HandlerFunc) {}

func (t *Context) Json(code int, data any) error {

	t.response.WriteHeader(code)
	t.response.Header().Set("Content-Type", "application/json")

	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if _, err := t.response.Write(b); err != nil {
		return err
	}

	return nil
}

func (t *Context) Context() context.Context {
	ctx := t.request.Context()
	return ctx
}

func (t *Context) Response() http.ResponseWriter {
	res := t.response
	return res
}

func (t *Context) Request() *http.Request {
	req := t.request
	return req
}

func (t *Context) parseRouter() {

	mux := t.ServeMux
	for key, route := range t.routes {
		r := route
		mux.HandleFunc(r.Path, func(writer http.ResponseWriter, request *http.Request) {
			if request.Method != r.Method {
				log.Printf("method not allow,path:%s,method:%s \n", r.Path, r.Method)
				return
			}

			defer func() {
				if err := recover(); err != nil {
					// panic handle
				}
			}()

			nt := beanqCtx.Get().(*Context)
			nt.response = writer
			nt.request = request

			if err := r.HandleFun(nt); err != nil {

			}

			nt.response = nil
			nt.request = nil
			beanqCtx.Put(nt)
		})
		t.routes[key] = nil
	}

}

func (t *Context) Run(port string) error {
	t.parseRouter()

	srv := &http.Server{
		Addr:    port,
		Handler: t.ServeMux,
	}
	log.Printf("web start,on port%s \n", port)
	return srv.ListenAndServe()
}
