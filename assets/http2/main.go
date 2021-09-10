package main

import (
	"fmt"
	"net/http"
	"os"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func main() {
	h2s := &http2.Server{}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor != 2 {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Must use HTTP/2 >:3")
		} else {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "Hello, %v, TLS: %v, Protocol: %v", r.URL.Path, r.TLS != nil, r.Proto)
		}
	})
	port := os.Getenv("PORT")
	server := &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%s", port),
		Handler: h2c.NewHandler(handler, h2s),
	}

	fmt.Printf("Listening [0.0.0.0:%s]...\n", port)
	err := server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
