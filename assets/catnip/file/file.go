package file

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"net/http"
	"net/url"
	"os"
)

func FileHandler(res http.ResponseWriter, req *http.Request) {
	filename := chi.URLParam(req, "filename")
	decodedFilename, err := url.PathUnescape(filename)
	if err != nil {
		http.Error(res, fmt.Sprintf("Cannot unescape file name: %s", filename), http.StatusBadRequest)
		return
	}

	_, err = os.Stat(decodedFilename)
	if err != nil {
		http.Error(res, http.StatusText(404) + ": " + decodedFilename, 404)
		return
	}

	content, err := os.ReadFile(decodedFilename)
	if err != nil {
		http.Error(res, http.StatusText(500) + ": " + err.Error(), 500)
		return
	}
	res.Write(append(content, '\n'))
}
