package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	instanceIndex := -1
	if indexStr := os.Getenv("CF_INSTANCE_INDEX"); indexStr != "" {
		if idx, err := strconv.Atoi(indexStr); err == nil {
			instanceIndex = idx
		}
	}

	if instanceIndex > 1 {
		fmt.Printf("Instance %d is quitting!\n", instanceIndex)
		os.Exit(1)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		instanceStr, ok := os.LookupEnv("CF_INSTANCE_INDEX")
		if !ok {
			instanceStr = "not defined"
		}
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Hello, you've reached the instance %s!", instanceStr)
	})

	fmt.Printf("Serving on port %s from instance %d\n", port, instanceIndex)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Could not start server: %v\n", err)
	}
}
