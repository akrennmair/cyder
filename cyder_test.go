package cyder

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

type Foo struct{}

func (f *Foo) Page(rr *RequestResponse) {
	fmt.Fprintf(rr, "called page")
}

func (f *Foo) Add(rr *RequestResponse, a, b int) {
	fmt.Fprintf(rr, "-%d-", a+b)
}

func (f *Foo) Bla(rr *RequestResponse, a string, b uint32, x float64) {
	fmt.Fprintf(rr, "-%s|%d-%.1f-", a, b, x)
}

func (f *Foo) Deliver403(rr *RequestResponse) {
	rr.StatusCode(403)
	fmt.Fprintf(rr, "delivered 403")
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
	foo := &Foo{}
	handler := Handler(foo)

	for _, test := range httphandler_test {
		resp := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "http://localhost:80"+test.url, nil)
		handler.ServeHTTP(resp, req)
		if resp.Code != test.respcode {
			t.Errorf("%s didn't deliver correct %d code; %d instead", test.url, test.respcode, resp.Code)
		}
		if resp.Body.String() != test.output {
			t.Errorf("%s didn't return correct content '%s'; '%s' instead", test.url, test.output, resp.Body.String())
		}
	}
}

func BenchmarkHTTPHandler(b *testing.B) {
	b.StopTimer()

	handler := Handler(&Foo{})

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		resp := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "http://localhost:80/bla/foobar/1234/0.1", nil)
		b.StartTimer()
		handler.ServeHTTP(resp, req)
	}
}
