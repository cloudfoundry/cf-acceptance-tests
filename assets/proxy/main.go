package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	systemPortString := os.Getenv("PORT")
	systemPort, err := strconv.Atoi(systemPortString)
	if err != nil {
		log.Fatal("invalid required env var PORT")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/proxy/", proxyHandler)
	mux.HandleFunc("/", infoHandler(systemPort))

	http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", systemPort), mux)
}

func infoHandler(port int) http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		addrs, err := net.InterfaceAddrs()
		if err != nil {
			panic(err)
		}
		addressStrings := []string{}
		for _, addr := range addrs {
			listenAddr := strings.Split(addr.String(), "/")[0]
			addressStrings = append(addressStrings, listenAddr)
		}

		respBytes, err := json.Marshal(struct {
			ListenAddresses []string
			Port            int
		}{
			ListenAddresses: addressStrings,
			Port:            port,
		})
		if err != nil {
			panic(err)
		}
		resp.Write(respBytes)
	}
}

func proxyHandler(resp http.ResponseWriter, req *http.Request) {
	destination := strings.TrimPrefix(req.URL.Path, "/proxy/")
	destination = "http://" + destination
	getResp, err := httpClient.Get(destination)
	if err != nil {
		fmt.Fprintf(os.Stderr, "request failed: %s", err)
		resp.WriteHeader(http.StatusInternalServerError)
		resp.Write([]byte(fmt.Sprintf("request failed: %s", err)))
		return
	}
	defer getResp.Body.Close()

	readBytes, err := ioutil.ReadAll(getResp.Body)
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		resp.Write([]byte(fmt.Sprintf("read body failed: %s", err)))
		return
	}

	resp.Write(readBytes)
}

var httpClient = &http.Client{
	Transport: &http.Transport{
		DisableKeepAlives: true,
		Dial: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 0,
		}).Dial,
	},
}
