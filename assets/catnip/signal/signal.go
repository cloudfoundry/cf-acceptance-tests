package signal

import (
	"net/http"
	"os"
)

func KillHandler(res http.ResponseWriter, req *http.Request) {
	os.Exit(1)
}
