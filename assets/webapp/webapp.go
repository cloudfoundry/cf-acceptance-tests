package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, err := w.Write([]byte("hi i am a standalone webapp"))
		if err != nil {
			panic(err)
		}
	})
	fmt.Println("ENV", os.Environ())
	port := os.Getenv("PORT")
	fmt.Println("Listening on port: ", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
