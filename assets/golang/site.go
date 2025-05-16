package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

type EndpointType struct {
	validationName string
	path           string
}

var endpointTypeMap = map[string]EndpointType{
	"api.ipify.org": {
		validationName: "IPv4",
		path:           "/ipv4-test",
	},
	"api6.ipify.org": {
		validationName: "IPv6",
		path:           "/ipv6-test",
	},
	"api64.ipify.org": {
		validationName: "Dual stack",
		path:           "/dual-stack-test",
	},
}

func main() {
	http.HandleFunc("/", handleRequest)
	http.HandleFunc("/requesturi/", echo)

	log.Printf("Starting server on %s\n", os.Getenv("PORT"))

	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", os.Getenv("PORT")),
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Could not start server: %s\n", err)
	}
}

func handleRequest(res http.ResponseWriter, req *http.Request) {
	for endpoint, data := range endpointTypeMap {
		if req.URL.Path == data.path {
			testEndpoint(res, endpoint, data.validationName)
			return
		}
	}

	hello(res, req)
}

func hello(res http.ResponseWriter, req *http.Request) {
	fmt.Fprintln(res, "Hello go, world")
}

func echo(res http.ResponseWriter, req *http.Request) {
	fmt.Fprintln(res, fmt.Sprintf("Request URI is [%s]\nQuery String is [%s]", req.RequestURI, req.URL.RawQuery))
}

func testEndpoint(res http.ResponseWriter, endpoint, validationName string) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(fmt.Sprintf("http://%s", endpoint))
	if err != nil {
		log.Printf("Failed to reach %s: %v\n", endpoint, err)
		writeTestResponse(res, validationName, false, "Unknown", err.Error())
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response: %v\n", err)
		writeTestResponse(res, validationName, false, "Unknown", err.Error())
		return
	}

	ipType := determineIPType(string(body))
	success := resp.StatusCode == http.StatusOK

	writeTestResponse(res, validationName, success, ipType, "")
}

func writeTestResponse(res http.ResponseWriter, validationName string, success bool, ipType, errorMsg string) {
	responseCode := http.StatusInternalServerError
	if success {
		responseCode = http.StatusOK
	}
	res.WriteHeader(responseCode)

	if errorMsg == "" {
		errorMsg = "none"
	}

	message := fmt.Sprintf("%s validation resulted in %s. Detected IP type is %s. Error message: %s.\n",
		validationName, map[bool]string{true: "success", false: "failure"}[success], ipType, errorMsg)
	res.Write([]byte(message))
}

func determineIPType(ipString string) string {
	ip := net.ParseIP(ipString)
	if ip == nil {
		return "Invalid IP"
	}

	if ip.To4() != nil {
		return "IPv4"
	}

	return "IPv6"
}