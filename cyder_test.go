package cyder

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"
)

type MockResponseWriter struct {
	StatusCode int
	Buffer     *bytes.Buffer
	header     http.Header
}

func NewMockResponseWriter() *MockResponseWriter {
	return &MockResponseWriter{Buffer: new(bytes.Buffer), header: make(http.Header)}
}

func (w *MockResponseWriter) Header() http.Header {
	return w.header
}

func (w *MockResponseWriter) WriteHeader(c int) {
	w.StatusCode = c
}

func (w *MockResponseWriter) Write(b []byte) (int, error) {
	if w.StatusCode == 0 {
		w.StatusCode = http.StatusOK
	}
	return w.Buffer.Write(b)
}

type Foo struct {
	Controller
}

func (f *Foo) Page() {
	fmt.Fprintf(f, "called page")
}

func (f *Foo) Add(a, b int) {
	fmt.Fprintf(f, "-%d-", a+b)
}

func (f *Foo) Bla(a string, b uint32, x float64) {
	fmt.Fprintf(f, "-%s|%d-%.1f-", a, b, x)
}

func TestHTTPHandler(t *testing.T) {
	resp := NewMockResponseWriter()
	foo := &Foo{}
	foo.setResponseWriter(resp)
	handler, err := newHTTPHandler(func() interface{} { return foo }, "/", "")
	if err != nil {
		t.Fatal(err)
	}

	req, _ := http.NewRequest("GET", "http://localhost:80/page", nil)

	handler.ServeHTTP(resp, req)
	if resp.StatusCode != http.StatusOK {
		t.Error("/page didn't deliver correct 200 code")
	}

	if resp.Buffer.String() != "called page" {
		t.Errorf("/page didn't return correct content; '%s' instead", resp.Buffer.String())
	}

	req, _ = http.NewRequest("GET", "http://localhost:80/add/23/42", nil)
	resp.Buffer.Reset()
	handler.ServeHTTP(resp, req)
	if resp.Buffer.String() != "-65-" {
		t.Errorf("/add/23/42 didn't return correct content; '%s' instead", resp.Buffer.String())
	}

	req, _ = http.NewRequest("GET", "http://localhost:80/bla/foobar/129374/3.5", nil)
	resp.Buffer.Reset()
	handler.ServeHTTP(resp, req)
	if resp.Buffer.String() != "-foobar|129374-3.5-" {
		t.Errorf("/bla/foobar/129374/3.5 didn't return correct content; '%s' instead", resp.Buffer.String())
	}
}
