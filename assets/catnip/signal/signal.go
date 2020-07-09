package signal

import (
	"net/http"
	"os"
)

func KillHandler(res http.ResponseWriter, req *http.Request) {
	currentProcess, _ := os.FindProcess(os.Getpid())
	currentProcess.Kill()
}
