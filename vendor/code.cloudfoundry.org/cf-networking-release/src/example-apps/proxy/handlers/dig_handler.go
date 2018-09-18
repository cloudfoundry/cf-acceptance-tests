package handlers

import (
	"net"
	"net/http"
	"strings"
	"encoding/json"
	"fmt"
	"os"
)

type DigHandler struct {
}

func (h *DigHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	destination := strings.TrimPrefix(req.URL.Path, "/dig/")
	destination = strings.Split(destination, ":")[0]

	ips, err := net.LookupIP(destination)
	if err != nil {
		handleDigError(err, destination, resp)
		return
	}

	var ip4s []string

	for _, ip := range ips {
		ip4s = append(ip4s, ip.To4().String())
	}

	ip4Json, err := json.Marshal(ip4s)
	if err != nil {
		handleDigError(err, destination, resp)
		return
	}

	resp.Write(ip4Json)
}


func handleDigError(err error, destination string, resp http.ResponseWriter) {
	msg := fmt.Sprintf("Failed to dig: %s: %s", destination, err)
	fmt.Fprintf(os.Stderr, msg)
	resp.WriteHeader(http.StatusInternalServerError)
	resp.Write([]byte(msg))
}