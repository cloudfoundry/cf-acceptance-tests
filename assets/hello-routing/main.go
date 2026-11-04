package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	server := &http.Server{
		Addr:    ":" + os.Getenv("PORT"),
		Handler: http.HandlerFunc(greet),
	}
	log.Fatal(server.ListenAndServe())
}

func greet(res http.ResponseWriter, req *http.Request) {
	var app struct {
		Name string `json:"name"`
	}
	if metadata := os.Getenv("VCAP_APPLICATION"); metadata != "" {
		if err := json.Unmarshal([]byte(metadata), &app); err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	fmt.Printf("%s %s %v\n", req.Method, req.URL, req.Header)
	res.Header().Set("Content-Type", "text/plain")
	http.SetCookie(res, &http.Cookie{Name: "JSESSIONID", Value: "12345"})
	fmt.Fprintf(res, "Hello, %s at index: %s!", app.Name, os.Getenv("CF_INSTANCE_INDEX"))
}
