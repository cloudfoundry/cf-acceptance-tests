package signal

import (
	"fmt"
	"net/http"
	"os"
)

func KillHandler(res http.ResponseWriter, req *http.Request) {
	currentProcess, _ := os.FindProcess(os.Getpid())
	currentProcess.Kill()
	fmt.Println("killing with os.Exit")
	os.Exit(1)
}
