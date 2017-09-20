package text

import (
	"io"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

func LargeHandler(res http.ResponseWriter, req *http.Request) {
	kbytes, _ := strconv.Atoi(mux.Vars(req)["kbytes"])

	k := make([]byte, 1024)
	for i := range k {
		k[i] = '1'
	}

	for i := 0; i < kbytes; i++ {
		io.WriteString(res, string(k))
	}
}
