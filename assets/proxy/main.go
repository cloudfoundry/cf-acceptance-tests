package main

import (
	"crypto/tls"
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
	mux.HandleFunc("/https_proxy/", httpsProxyHandler)
	mux.HandleFunc("/", infoHandler(systemPort))

	_ = http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", systemPort), mux)
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
		_, _ = resp.Write(respBytes)
	}
}

func proxyHandler(resp http.ResponseWriter, req *http.Request) {
	destination := strings.TrimPrefix(req.URL.Path, "/proxy/")
	destination = fmt.Sprintf("%s://%s", "http", destination)
	handleRequest(destination, resp, req)
}

func httpsProxyHandler(resp http.ResponseWriter, req *http.Request) {
	destination := strings.TrimPrefix(req.URL.Path, "/https_proxy/")
	destination = fmt.Sprintf("%s://%s", "https", destination)
	handleRequest(destination, resp, req)
}

func handleRequest(destination string, resp http.ResponseWriter, req *http.Request) {
	getResp, err := httpClient.Get(destination)
	if err != nil {
		fmt.Fprintf(os.Stderr, "request failed: %s", err)
		resp.WriteHeader(http.StatusInternalServerError)
		_, _ = resp.Write([]byte(fmt.Sprintf("request failed: %s", err)))
		return
	}
	defer getResp.Body.Close()

	readBytes, err := ioutil.ReadAll(getResp.Body)
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		_, _ = resp.Write([]byte(fmt.Sprintf("read body failed: %s", err)))
		return
	}

	_, _ = resp.Write(readBytes)
}

var httpClient = &http.Client{
	Transport: &http.Transport{
		DisableKeepAlives: true,
		Dial: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 0,
		}).Dial,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	},
}
