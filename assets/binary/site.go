package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

func main() {
	http.HandleFunc("/", hello)
	http.HandleFunc("/env", env)
	fmt.Println("listening...")
	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", os.Getenv("PORT")),
		Handler: nil,
	}
	err := server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}

func hello(res http.ResponseWriter, _ *http.Request) {
	fmt.Fprintln(res, "Hello from a binary")
}

func env(res http.ResponseWriter, _ *http.Request) {
	envVariables := make(map[string]string)
	for _, envKeyValue := range os.Environ() {
		keyValue := strings.Split(envKeyValue, "=")
		envVariables[keyValue[0]] = keyValue[1]
	}
	envJsonBytes, err := json.Marshal(envVariables)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
	fmt.Fprintln(res, string(envJsonBytes))
}
