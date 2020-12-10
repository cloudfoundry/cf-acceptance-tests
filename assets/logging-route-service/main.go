package main

import (
	"bytes"
	"crypto/tls"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"time"
)

const (
	DEFAULT_PORT              = "8080"
	CF_FORWARDED_URL_HEADER   = "X-Cf-Forwarded-Url"
	CF_PROXY_SIGNATURE_HEADER = "X-Cf-Proxy-Signature"
)

func main() {
	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = DEFAULT_PORT
	}
	skipSslValidation, _ := strconv.ParseBool(os.Getenv("SKIP_SSL_VALIDATION"))

	log.SetOutput(os.Stdout)

	roundTripper := NewLoggingRoundTripper(skipSslValidation)
	proxy := NewProxy(roundTripper, skipSslValidation)

	log.Fatal(http.ListenAndServe(":"+port, proxy))
}

func NewProxy(transport http.RoundTripper, skipSslValidation bool) http.Handler {
	reverseProxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			forwardedURL := req.Header.Get(CF_FORWARDED_URL_HEADER)
			sigHeader := req.Header.Get(CF_PROXY_SIGNATURE_HEADER)

			var body []byte
			var err error
			if req.Body != nil {
				body, err = ioutil.ReadAll(req.Body)
				if err != nil {
					log.Fatalln(err.Error())
				}
				req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
			}
			logRequest(forwardedURL, sigHeader, string(body), req.Header, skipSslValidation)

			err = sleep()
			if err != nil {
				log.Fatalln(err.Error())
			}

			// Note that url.Parse is decoding any url-encoded characters.
			url, err := url.Parse(forwardedURL)
			if err != nil {
				log.Fatalln(err.Error())
			}

			req.URL = url
			req.Host = url.Host
		},
		Transport: transport,
	}
	return reverseProxy
}

func logRequest(forwardedURL, sigHeader, body string, headers http.Header, skipSslValidation bool) {
	log.Printf("Skip ssl validation set to %t", skipSslValidation)
	log.Println("Received request: ")
	log.Printf("%s: %s\n", CF_FORWARDED_URL_HEADER, forwardedURL)
	log.Printf("%s: %s\n", CF_PROXY_SIGNATURE_HEADER, sigHeader)
	log.Println("")
	log.Printf("Headers: %#v\n", headers)
	log.Println("")
	log.Printf("Request Body: %s\n", body)
}

type LoggingRoundTripper struct {
	transport http.RoundTripper
}

func NewLoggingRoundTripper(skipSslValidation bool) *LoggingRoundTripper {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: skipSslValidation},
	}
	return &LoggingRoundTripper{
		transport: tr,
	}
}

func (lrt *LoggingRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	var err error
	var res *http.Response

	log.Printf("Forwarding to: %s\n", request.URL.String())
	res, err = lrt.transport.RoundTrip(request)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatalln(err.Error())
	}
	log.Println("")
	log.Printf("Response Headers: %#v\n", res.Header)
	log.Println("")
	log.Printf("Response Body: %s\n", string(body))
	log.Println("")
	res.Body = ioutil.NopCloser(bytes.NewBuffer(body))

	log.Println("Sending response to GoRouter...")

	return res, err
}

func sleep() error {
	sleepMilliString := os.Getenv("ROUTE_SERVICE_SLEEP_MILLI")
	if sleepMilliString != "" {
		sleepMilli, err := strconv.ParseInt(sleepMilliString, 0, 64)
		if err != nil {
			return err
		}

		log.Printf("Sleeping for %d milliseconds\n", sleepMilli)
		time.Sleep(time.Duration(sleepMilli) * time.Millisecond)

	}
	return nil
}
