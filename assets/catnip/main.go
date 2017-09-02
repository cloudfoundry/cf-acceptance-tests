package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"

	"github.com/cloudfoundry/cf-acceptance-tests/assets/catnip/env"
)

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/", HomeHandler)

	r.HandleFunc("/env.json", env.JsonHandler)
	envRouter := r.PathPrefix("/env").Subrouter()
	envRouter.HandleFunc("/{name}", env.NameHandler)

	fmt.Printf("listening on port %s...\n", os.Getenv("PORT"))
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", os.Getenv("PORT")), r))
}

func HomeHandler(res http.ResponseWriter, req *http.Request) {
	io.WriteString(res, "Hi, I'm Dora!")
}
