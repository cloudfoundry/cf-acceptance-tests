package handlers

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
)

type InfoHandler struct {
	Port int
}

func (h *InfoHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
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
		Port:            h.Port,
	})
	if err != nil {
		panic(err)
	}
	resp.Write(respBytes)
	return
}
