package handlers

import (
	"net/http"
)

type EchoSourceIPHandler struct{}

func (h *EchoSourceIPHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	resp.Write([]byte(req.RemoteAddr))
	return
}
