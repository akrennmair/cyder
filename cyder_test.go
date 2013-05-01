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

type Foo struct { }

func (f *Foo) Page(rr *RequestResponse) {
	fmt.Fprintf(rr.W, "called page")
}

func (f *Foo) Add(rr *RequestResponse, a, b int) {
	fmt.Fprintf(rr.W, "-%d-", a+b)
}

func (f *Foo) Bla(rr *RequestResponse, a string, b uint32, x float64) {
	fmt.Fprintf(rr.W, "-%s|%d-%.1f-", a, b, x)
}

func (f *Foo) Deliver403(rr *RequestResponse) {
	rr.W.WriteHeader(403)
	fmt.Fprintf(rr.W, "delivered 403")
}

var httphandler_test = []struct {
	url      string
	respcode int
	output   string
}{
	{"/page", http.StatusOK, "called page"},
	{"/add/23/42", http.StatusOK, "-65-"},
	{"/bla/foobar/129374/3.5", http.StatusOK, "-foobar|129374-3.5-"},
	{"/deliver403", http.StatusForbidden, "delivered 403"},
}

func TestHTTPHandler(t *testing.T) {
	resp := NewMockResponseWriter()
	foo := &Foo{}
	handler := Handler(foo)

	for _, test := range httphandler_test {
		resp.Buffer.Reset()
		req, _ := http.NewRequest("GET", "http://localhost:80"+test.url, nil)
		handler.ServeHTTP(resp, req)
		if resp.StatusCode != test.respcode {
			t.Errorf("%s didn't deliver correct %d code; %d instead", test.url, test.respcode, resp.StatusCode)
		}
		if resp.Buffer.String() != test.output {
			t.Errorf("%s didn't return correct content '%s'; '%s' instead", test.url, test.output, resp.Buffer.String())
		}
	}

}

func BenchmarkHTTPHandler(b *testing.B) {
	b.StopTimer()

	resp := NewMockResponseWriter()
	foo := &Foo{}
	handler := Handler(foo)

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		resp.Buffer.Reset()
		req, _ := http.NewRequest("GET", "http://localhost:80/bla/foobar/1234/0.1", nil)
		b.StartTimer()
		handler.ServeHTTP(resp, req)
	}
}
