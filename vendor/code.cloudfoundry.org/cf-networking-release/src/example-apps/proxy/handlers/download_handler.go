package handlers

import (
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
)

type DownloadHandler struct{}

func (h *DownloadHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	requestBytes := strings.TrimPrefix(req.URL.Path, "/download/")
	numBytes, err := strconv.Atoi(requestBytes)
	if err != nil || numBytes < 0 {
		resp.WriteHeader(http.StatusInternalServerError)
		resp.Write([]byte(fmt.Sprintf("requested number of bytes must be a positive integer, got: %s", requestBytes)))
		return
	}

	respBytes := make([]byte, numBytes)
	rand.Read(respBytes)
	resp.Write(respBytes)
}
