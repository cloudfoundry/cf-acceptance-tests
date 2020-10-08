package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

func main() {
	listenAddress := fmt.Sprintf(":%s", os.Getenv("PORT"))
	http.HandleFunc("/", handleHTTP)
	panic(http.ListenAndServe(listenAddress, nil))
}

func handleHTTP(resp http.ResponseWriter, req *http.Request) {
	msg, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(resp, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println(string(msg))
}
