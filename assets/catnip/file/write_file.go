package file

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/go-chi/chi/v5"
)

func WriteFileHandler(res http.ResponseWriter, req *http.Request) {
	filename := chi.URLParam(req, "filename")
	decodedFilename, err := url.PathUnescape(filename)
	if err != nil {
		http.Error(res, fmt.Sprintf("Cannot unescape file name: %s", filename), http.StatusBadRequest)
		return
	}

	decodedFilename = "/" + decodedFilename

	content, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(res, fmt.Sprintf("failed to read the file: %v", err), http.StatusInternalServerError)
		return
	}

	err = os.WriteFile(decodedFilename, content, 0644)
	if err != nil {
		http.Error(res, http.StatusText(500)+": "+err.Error(), 500)
		return
	}

	res.WriteHeader(http.StatusCreated)
}
