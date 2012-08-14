package cyder

import (
	"errors"
	"io"
	"fmt"
	"net/http"
	"reflect"
	"strings"
)


type Application struct {
	mux        *http.ServeMux
	server     *http.Server
	htdocs_dir string
}

func NewApplication() *Application {
	app := &Application{}
	app.mux = http.NewServeMux()
	return app
}

func (app *Application) SetHtdocs(htdocs string) {
	app.htdocs_dir = htdocs
}

func (app *Application) RegisterController(prefix string, generator func() interface{}) error {
	if prefix == "" || prefix[0:1] != "/" || prefix[len(prefix)-1:] != "/" {
		return errors.New("invalid prefix (must be non-empty and begin and end with /)")
	}
	if generator == nil {
		return errors.New("invalid nil generator")
	}

	if handler, err := newHTTPHandler(generator, prefix, app.htdocs_dir); err != nil {
		return err
	} else {
		app.mux.Handle(prefix, http.StripPrefix(prefix[0:len(prefix)-1], handler))
	}
	return nil
}

func (app *Application) ListenAndServe(addr string) error {
	app.server = &http.Server{Handler: app.mux, Addr: addr}
	return app.server.ListenAndServe()
}

func (app *Application) ListenAndServeTLS(addr, certfile, keyfile string) error {
	app.server = &http.Server{Handler: app.mux, Addr: addr}
	return app.server.ListenAndServeTLS(certfile, keyfile)
}

type Controller struct {
	rw http.ResponseWriter
}

type httpController interface {
	Header() http.Header
	WriteHeader(int)
	setResponseWriter(http.ResponseWriter)
	io.Writer
}

func (c *Controller) setResponseWriter(w http.ResponseWriter) {
	c.rw = w
}

func (c *Controller) Header() http.Header {
	return c.rw.Header()
}

func (c *Controller) WriteHeader(code int) {
	c.rw.WriteHeader(code)
}

func (c *Controller) Write(data []byte) (int, error) {
	return c.rw.Write(data)
}

type httpHandler struct {
	methods   map[string]reflect.Method
	generator func() interface{}
	htdocs    string
	fileserver http.Handler
}

func newHTTPHandler(gen func() interface{}, prefix, htdocs string) (*httpHandler, error) {
	handler := &httpHandler{generator: gen, methods: make(map[string]reflect.Method)}
	receiver := handler.generator()

	if _, ok := receiver.(httpController); !ok {
		return nil, errors.New("generator doesn't produce cyder.Controller elements")
	}

	t := reflect.TypeOf(receiver)

	if htdocs != "" {
		handler.fileserver = http.FileServer(http.Dir(htdocs + prefix))
	}

	for i := 0; i < t.NumMethod(); i++ {
		method := t.Method(i)
		if method.PkgPath == "" {
			handler.methods[strings.ToLower(method.Name)] = method
		}
	}

	return handler, nil
}

func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	method := r.URL.Path[1:]
	if method == "" {
		method = "index"
	}
	fmt.Printf("calling %s\n", method)
	m, ok := h.methods[method]
	if !ok {
		if h.fileserver != nil {
			h.fileserver.ServeHTTP(w, r)
		} else {
			fmt.Printf("method %s not found\n", method)
		}
		return
	}
	receiver := h.generator()

	ctrl, _ := receiver.(httpController)
	ctrl.setResponseWriter(w)

	m.Func.Call([]reflect.Value{reflect.ValueOf(receiver)})
}
