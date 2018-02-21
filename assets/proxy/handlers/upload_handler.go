package handlers

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

type UploadHandler struct{}

func (h *UploadHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	if req.Body == nil {
		resp.Write([]byte("0 bytes received and read"))
		return
	}
	bodyBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		resp.Write([]byte(fmt.Sprintf("error: %s", err)))
		return
	}

	resp.Write([]byte(fmt.Sprintf("%d bytes received and read", len(bodyBytes))))
}
