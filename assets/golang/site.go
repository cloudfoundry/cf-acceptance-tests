package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

var endpointMap = map[string]string{
	"/ipv4-test":     "https://api.ipify.org",
	"/ipv6-test":     "https://api6.ipify.org",
	"/dual-stack-test": "https://api64.ipify.org",
}

func main() {
	http.HandleFunc("/", hello)
	http.HandleFunc("/requesturi/", echo)


	for path, apiURL := range endpointMap {
		http.HandleFunc(path, createIPHandler(apiURL))
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Listening on port %s...\n", port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatalf("Could not start server: %v\n", err)
	}
}

func hello(res http.ResponseWriter, req *http.Request) {
	fmt.Fprintln(res, "Hello go, world")
}

func echo(res http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(res, "Request URI is [%s]\nQuery String is [%s]\n", req.RequestURI, req.URL.RawQuery)
}

func createIPHandler(apiURL string) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		fetchAndWriteIP(res, apiURL)
	}
}

func fetchAndWriteIP(res http.ResponseWriter, apiURL string) {
	client := &http.Client{Timeout: 5 * time.Second}

	resp, err := client.Get(apiURL)
	if err != nil {
		log.Printf("Error fetching from %s: %v\n", apiURL, err)
		res.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(res, "Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response: %v\n", err)
		res.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(res, "Error: %v\n", err)
		return
	}

	res.WriteHeader(resp.StatusCode)
	fmt.Fprintln(res, string(body))
}
