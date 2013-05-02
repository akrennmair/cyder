package cyder

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

const (
	VERSION      = "0.1"
	CYDER_STRING = "cyder/" + VERSION
)

const (
	OPTIONS = 1 << iota
	GET
	HEAD
	POST
	PUT
	DELETE
	TRACE
	CONNECT
)

type httpHandler struct {
	methods map[int]map[string]reflect.Method
	ctrl    interface{}
}

func Handler(ctrl interface{}) http.Handler {
	t := reflect.TypeOf(ctrl)

	methods := make(map[int]map[string]reflect.Method)
	for i := OPTIONS; i <= CONNECT; i++ {
		methods[i] = make(map[string]reflect.Method)
	}

	handler := &httpHandler{methods: methods, ctrl: ctrl}

	rrArgType := reflect.TypeOf(&RequestResponse{})

	for i := 0; i < t.NumMethod(); i++ {
		method := t.Method(i)
		if method.PkgPath == "" {
			if method.Type.NumIn() < 2 || method.Type.In(1) != rrArgType {
				// needs to have at least 2 arguments (receiver and *RequestResponse), otherwise we ignore it.
				// also, if second argument is not a *RequestResponse, ignore method.
				continue
			}
			if methodName, httpMethod, found := findMethod(method.Name); found {
				handler.methods[httpMethod][methodName] = method
			} else {
				for i := OPTIONS; i <= CONNECT; i++ {
					handler.methods[i][methodName] = method
				}
			}
		}
	}

	return handler
}

var httpMethods = []string{"OPTIONS", "GET", "HEAD", "POST", "PUT", "DELETE", "TRACE", "CONNECT"}

func findMethod(method string) (methodName string, httpMethod int, found bool) {
	for i, m := range httpMethods {
		if len(method) > len(m) {
			if strings.ToUpper(method[:len(m)]) == m {
				return strings.ToLower(method[len(m):]), i + 1, true
			}
		}
	}
	return strings.ToLower(method), 0, false
}

func getHTTPMethod(httpMethod string) int {
	for i, m := range httpMethods {
		if m == httpMethod {
			return i + 1
		}
	}
	return 0
}

type RequestResponse struct {
	ResponseWriter http.ResponseWriter
	Request        *http.Request
	statusCode     int
	sentHeaders    bool
	parsedForm     bool
}

func (rr *RequestResponse) StatusCode(statusCode int) {
	if !rr.sentHeaders {
		rr.statusCode = statusCode
	}
}

func (rr *RequestResponse) SetHeader(name, value string) {
	if !rr.sentHeaders {
		rr.ResponseWriter.Header()[name] = []string{value}
	}
}

func (rr *RequestResponse) ContentType(value string) {
	rr.SetHeader("Content-Type", value)
}

func (rr *RequestResponse) ContentLength(length uint64) {
	rr.SetHeader("Content-Length", fmt.Sprintf("%d", length))
}

func (rr *RequestResponse) Write(data []byte) (int, error) {
	if !rr.sentHeaders {
		rr.ResponseWriter.WriteHeader(rr.statusCode)
		rr.sentHeaders = true
	}
	return rr.ResponseWriter.Write(data)
}

func (rr *RequestResponse) WriteString(data string) (int, error) {
	return rr.Write([]byte(data))
}

func (rr *RequestResponse) WriteJSON(v interface{}) error {
	rr.ContentType("application/json")
	enc := json.NewEncoder(rr)
	return enc.Encode(v)
}

func (rr *RequestResponse) Form() url.Values {
	if !rr.parsedForm {
		rr.Request.ParseForm() // TODO: handle error?
		rr.parsedForm = true
	}
	return rr.Request.Form
}

func (rr *RequestResponse) ParseJSON(v interface{}) error {
	if rr.Request.Method != "POST" && rr.Request.Method != "PUT" {
		return fmt.Errorf("can't use ParseJSON with HTTP method %s", rr.Request.Method)
	}
	if len(rr.Request.Header["Content-Type"]) == 0 || rr.Request.Header["Content-Type"][0] != "application/json" {
		return errors.New("Content-Type is not application/json")
	}
	dec := json.NewDecoder(rr.Request.Body)
	return dec.Decode(v)
}

func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path_elements := strings.Split(r.URL.Path[1:], "/")

	methodName := path_elements[0]
	path_elements = path_elements[1:]

	if methodName == "" {
		methodName = "index"
	}

	httpMethod := getHTTPMethod(r.Method)

	m, ok := h.methods[httpMethod][methodName]
	if !ok {
		http.Error(w, fmt.Sprintf("page %s not found", methodName), http.StatusNotFound)
		return
	}

	if len(path_elements)+2 != m.Type.NumIn() {
		log.Printf("expected %d arguments, got %d.", m.Type.NumIn(), len(path_elements))
		http.Error(w, "incorrect number of arguments", http.StatusBadRequest)
		return
	}

	args := []reflect.Value{reflect.ValueOf(h.ctrl), reflect.ValueOf(&RequestResponse{ResponseWriter: w, Request: r})}

	for i, arg := range path_elements {
		funcarg := m.Type.In(i + 2)
		newarg, err := convertArgument(arg, funcarg)
		if err != nil {
			log.Printf("%s: converting arg %d (%s) failed: %v", methodName, i, arg, err)
			http.Error(w, fmt.Sprintf("incorrect argument %d", i+1), http.StatusBadRequest)
			return
		}
		args = append(args, newarg)
	}

	w.Header()["X-Powered-By"] = []string{CYDER_STRING}

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
