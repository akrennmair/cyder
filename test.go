package main

import (
	"./cyder"
	"fmt"
)

type Foo struct {
	cyder.Controller
}

func (f *Foo) Index() {
	f.WriteHeader(200)
	fmt.Fprintf(f, "hello world from index")
}

func (f *Foo) Foobar() {
	f.WriteHeader(200)
	fmt.Fprintf(f, "hello world from foobar")
}

func (f *Foo) Pope(a int) {
	f.WriteHeader(200)
	fmt.Fprintf(f, "Pope: %d", a)
}

type Index struct {
	cyder.Controller
}

func (i *Index) Index() {
	i.WriteHeader(200)
	fmt.Fprintf(i, "hello world from /")
}

func main() {
	app := cyder.NewApplication()
	app.SetHtdocs("htdocs")

	if err := app.RegisterController("/foo/", func() interface{} { return &Foo{} }); err != nil {
		fmt.Printf("RegisterController /foo/ failed: %v\n", err)
		return
	}

	if err := app.RegisterController("/", func() interface{} { return &Index{} }); err != nil {
		fmt.Printf("RegisterController / failed: %v\n", err)
		return
	}

	fmt.Printf("Starting HTTP server on :8000...\n")
	if err := app.ListenAndServe(":8000"); err != nil {
		fmt.Printf("ListenAndServe: %v\n", err)
	}
}
