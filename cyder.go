package cyder

import (
	"errors"
	"io"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
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
	path_elements := strings.Split(r.URL.Path[1:], "/")

	method := path_elements[0]
	path_elements = path_elements[1:]

	if method == "" {
		method = "index"
	}

	m, ok := h.methods[method]
	if !ok {
		if h.fileserver != nil {
			h.fileserver.ServeHTTP(w, r)
		} else {
			http.Error(w, fmt.Sprintf("page %s not found", method), http.StatusNotFound)
		}
		return
	}

	if len(path_elements)+1 != m.Type.NumIn() {
		http.Error(w, "incorrect number of arguments", http.StatusBadRequest)
		return
	}

	receiver := h.generator()

	args := []reflect.Value{reflect.ValueOf(receiver)}

	for i, arg := range path_elements {
		funcarg := m.Type.In(i+1)
		newarg, err := convertArgument(arg, funcarg)
		if err != nil {
			http.Error(w, "incorrect argument", http.StatusBadRequest)
			return
		}
		args = append(args, newarg)
	}

	ctrl, _ := receiver.(httpController)
	ctrl.setResponseWriter(w)

	m.Func.Call(args)
}

func convertArgument(arg string, argtype reflect.Type) (v reflect.Value, err error) {
	p := reflect.New(reflect.PtrTo(argtype))
	v = p.Elem()
	switch argtype.Kind() {
	case reflect.Bool:
		b := false
		if arg == "true" {
			b = true
		} else if arg == "false" {
			b = false
		} else {
			return v, fmt.Errorf("'%s' is not a valid bool", arg)
		}
		v = reflect.ValueOf(&b).Elem()

	case reflect.Int:
		i, err := strconv.ParseInt(arg, 10, 32)
		if err != nil {
			return v, err
		}
		ii := int(i)
		v = reflect.ValueOf(&ii).Elem()

	case reflect.Int8:
		i, err := strconv.ParseInt(arg, 10, 8)
		if err != nil {
			return v, err
		}
		ii := int8(i)
		v = reflect.ValueOf(&ii).Elem()

	case reflect.Int16:
		i, err := strconv.ParseInt(arg, 10, 16)
		if err != nil {
			return v, err
		}
		ii := int16(i)
		v = reflect.ValueOf(&ii).Elem()

	case reflect.Int32:
		i, err := strconv.ParseInt(arg, 10, 32)
		if err != nil {
			return v, err
		}
		ii := int32(i)
		v = reflect.ValueOf(&ii).Elem()

	case reflect.Int64:
		i, err := strconv.ParseInt(arg, 10, 64)
		if err != nil {
			return v, err
		}
		v = reflect.ValueOf(&i).Elem()

	case reflect.Uint:
		i, err := strconv.ParseUint(arg, 10, 32)
		if err != nil {
			return v, err
		}
		ii := uint(i)
		v = reflect.ValueOf(&ii).Elem()

	case reflect.Uint8:
		i, err := strconv.ParseUint(arg, 10, 8)
		if err != nil {
			return v, err
		}
		ii := uint8(i)
		v = reflect.ValueOf(&ii).Elem()

	case reflect.Uint16:
		i, err := strconv.ParseUint(arg, 10, 16)
		if err != nil {
			return v, err
		}
		ii := uint16(i)
		v = reflect.ValueOf(&ii).Elem()

	case reflect.Uint32:
		i, err := strconv.ParseUint(arg, 10, 32)
		if err != nil {
			return v, err
		}
		ii := uint32(i)
		v = reflect.ValueOf(&ii).Elem()

	case reflect.Uint64:
		i, err := strconv.ParseUint(arg, 10, 64)
		if err != nil {
			return v, err
		}
		v = reflect.ValueOf(&i).Elem()

	case reflect.String:
		v = reflect.ValueOf(&arg).Elem()

	case reflect.Float32: 
		f, err := strconv.ParseFloat(arg, 32)
		if err != nil {
			return v, err
		}
		ff := float32(f)
		v = reflect.ValueOf(&ff).Elem()
	
	case reflect.Float64:
		f, err := strconv.ParseFloat(arg, 64)
		if err != nil {
			return v, err
		}
		v = reflect.ValueOf(&f).Elem()

	default:
		return v, errors.New("unsupported argument type")
	}
	return v, nil
}
