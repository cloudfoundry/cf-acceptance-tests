package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
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
		log.Fatal("Invalid required env var PORT")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/proxy/", ipv6ProxyHandler)
	mux.HandleFunc("/https_proxy/", ipv6HttpsProxyHandler)
	mux.HandleFunc("/", ipv6InfoHandler(systemPort))

	server := &http.Server{
		Addr:    fmt.Sprintf("[::]:%d", systemPort), // Listen on IPv6 interfaces
		Handler: mux,
	}
	_ = server.ListenAndServe()
}

func ipv6InfoHandler(port int) http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		addrs, err := net.InterfaceAddrs()
		if err != nil {
			panic(err)
		}
		addressStrings := []string{}
		for _, addr := range addrs {
			ip, _, _ := net.ParseCIDR(addr.String()) 
			if ip.To4() == nil { // Ensure it's not IPv4
				addressStrings = append(addressStrings, ip.String())
			}
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

func ipv6ProxyHandler(resp http.ResponseWriter, req *http.Request) {
	destination := strings.TrimPrefix(req.URL.Path, "/proxy/")
	destination = formatIPv6Destination("http", destination)
	handleRequest(destination, resp, req)
}

func ipv6HttpsProxyHandler(resp http.ResponseWriter, req *http.Request) {
	destination := strings.TrimPrefix(req.URL.Path, "/https_proxy/")
	destination = formatIPv6Destination("https", destination)
	handleRequest(destination, resp, req)
}

func formatIPv6Destination(proto, destination string) string {
	if strings.Contains(destination, ":") {
		destination = fmt.Sprintf("[%s]", destination) // Encapsulate IPv6 addresses
	}
	return fmt.Sprintf("%s://%s", proto, destination)
}

func handleRequest(destination string, resp http.ResponseWriter, req *http.Request) {
	getResp, err := httpClient.Get(destination)
	if err != nil {
		fmt.Fprintf(os.Stderr, "request failed: %s\n", err)
		resp.WriteHeader(http.StatusInternalServerError)
		_, _ = resp.Write([]byte(fmt.Sprintf("request failed: %s", err)))
		return
	}
	defer getResp.Body.Close()

	readBytes, err := io.ReadAll(getResp.Body)
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
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 0,
		}).DialContext,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	},
}

