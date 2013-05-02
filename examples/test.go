package main

import (
	"fmt"
	"github.com/akrennmair/cyder"
	"net/http"
)

type Foo struct{}

func (f *Foo) Index(rr *cyder.RequestResponse) {
	rr.StatusCode(200)
	fmt.Fprintf(rr, "hello world from index")
}

func (f *Foo) Foobar(rr *cyder.RequestResponse) {
	rr.StatusCode(200)
	fmt.Fprintf(rr, "hello world from foobar")
}

func (f *Foo) Pope(rr *cyder.RequestResponse, a int) {
	rr.StatusCode(200)
	fmt.Fprintf(rr, "Pope: %d", a)
}

type Index struct{}

func (i *Index) Index(rr *cyder.RequestResponse) {
	rr.StatusCode(200)
	fmt.Fprintf(rr, "hello world from /")
}

func main() {
	http.Handle("/foo/", http.StripPrefix("/foo", cyder.Handler(&Foo{})))
	http.Handle("/", cyder.Handler(&Index{}))

	fmt.Printf("Starting HTTP server on :8000...\n")
	if err := http.ListenAndServe(":8000", nil); err != nil {
		fmt.Printf("ListenAndServe: %v\n", err)
	}
}
