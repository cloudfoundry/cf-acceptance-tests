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
		log.Fatal("invalid required env var PORT")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/proxy/", proxyHandler)
	mux.HandleFunc("/https_proxy/", httpsProxyHandler)
	mux.HandleFunc("/mtls_proxy/", mtlsProxyHandler)
	mux.HandleFunc("/headers", headersHandler)
	mux.HandleFunc("/", infoHandler(systemPort))

	server := &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%d", systemPort),
		Handler: mux,
	}
	_ = server.ListenAndServe()
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

	readBytes, err := io.ReadAll(getResp.Body)
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		_, _ = resp.Write([]byte(fmt.Sprintf("read body failed: %s", err)))
		return
	}

	_, _ = resp.Write(readBytes)
}

func headersHandler(resp http.ResponseWriter, req *http.Request) {
	headers := make(map[string]string)
	for name, values := range req.Header {
		headers[name] = strings.Join(values, ", ")
	}
	resp.Header().Set("Content-Type", "application/json")
	json.NewEncoder(resp).Encode(headers)
}

func mtlsProxyHandler(resp http.ResponseWriter, req *http.Request) {
	destination := strings.TrimPrefix(req.URL.Path, "/mtls_proxy/")
	destination = fmt.Sprintf("https://%s", destination)

	certFile := os.Getenv("CF_INSTANCE_CERT")
	keyFile := os.Getenv("CF_INSTANCE_KEY")
	if certFile == "" || keyFile == "" {
		resp.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(resp).Encode(map[string]interface{}{
			"status":      "error",
			"status_code": 500,
			"error":       "CF_INSTANCE_CERT or CF_INSTANCE_KEY not set",
		})
		return
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(resp).Encode(map[string]interface{}{
			"status":      "error",
			"status_code": 500,
			"error":       fmt.Sprintf("failed to load client cert: %s", err),
		})
		return
	}

	client := &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true,
			Dial: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 0,
			}).Dial,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
				Certificates:      []tls.Certificate{cert},
			},
		},
	}

	getResp, err := client.Get(destination)
	if err != nil {
		resp.Header().Set("Content-Type", "application/json")
		json.NewEncoder(resp).Encode(map[string]interface{}{
			"status":      "error",
			"status_code": 0,
			"error":       fmt.Sprintf("request failed: %s", err),
		})
		return
	}
	defer getResp.Body.Close()

	body, err := io.ReadAll(getResp.Body)
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(resp).Encode(map[string]interface{}{
			"status":      "error",
			"status_code": 0,
			"error":       fmt.Sprintf("read body failed: %s", err),
		})
		return
	}

	respHeaders := make(map[string]string)
	for name, values := range getResp.Header {
		respHeaders[name] = strings.Join(values, ", ")
	}

	resp.Header().Set("Content-Type", "application/json")
	json.NewEncoder(resp).Encode(map[string]interface{}{
		"status":      "success",
		"status_code": getResp.StatusCode,
		"body":        string(body),
		"headers":     respHeaders,
	})
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
