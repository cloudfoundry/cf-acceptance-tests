package handlers

import (
	"mime/multipart"
	"net/http"

	"github.com/cloudfoundry/gosteno"
)

type httpHandler struct {
	messages <-chan []byte
	logger   *gosteno.Logger
}

func NewHttpHandler(m <-chan []byte, logger *gosteno.Logger) *httpHandler {
	return &httpHandler{messages: m, logger: logger}
}

func (h *httpHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	h.logger.Debugf("http handler: ServeHTTP entered with request %v", r.URL)
	defer h.logger.Debugf("http handler: ServeHTTP exited")

	mp := multipart.NewWriter(rw)
	defer mp.Close()

	rw.Header().Set("Content-Type", `multipart/x-protobuf; boundary=`+mp.Boundary())

	for message := range h.messages {
		partWriter, err := mp.CreatePart(nil)
		if err != nil {
			h.logger.Infof("http handler: Client went away while serving recent logs")
			return
		}

		partWriter.Write(message)
	}
}
