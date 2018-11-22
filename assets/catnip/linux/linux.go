package linux

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"syscall"

	"github.com/gorilla/mux"
)

func ReleaseHandler(res http.ResponseWriter, req *http.Request) {
	cmd := exec.Command("lsb_release", "--all")
	outBytes, _ := cmd.Output()

	res.Write(outBytes)
}

func MyIPHandler(res http.ResponseWriter, req *http.Request) {
	// example output from `ip`: `1.0.0.0 via 169.254.0.1 dev eth0 src 10.255.97.224 uid 2000`,
	// in older rootfs the `uid 2000` is omitted
	cmd := exec.Command("bash", "-c", `ip route get 1 | sed -n -e 's/^.*src \([^ ]\+\).*$/\1/p'`)
	outBytes, _ := cmd.Output()

	res.Write(outBytes)
}

func CurlHandler(res http.ResponseWriter, req *http.Request) {
	host := mux.Vars(req)["host"]
	port := mux.Vars(req)["port"]
	if port == "" {
		port = "80"
	}

	cmd := exec.Command("curl", "-m", "3", "-v", "-i", fmt.Sprintf("%s:%s", host, port))
	outBuf := bytes.NewBuffer([]byte{})
	errBuf := bytes.NewBuffer([]byte{})
	cmd.Stdout = outBuf
	cmd.Stderr = errBuf

	err := cmd.Run()

	exitCode := 0
	if e, ok := err.(*exec.ExitError); ok {
		exitCode = e.ProcessState.Sys().(syscall.WaitStatus).ExitStatus()
	}

	curlOutput := struct {
		Stdout     string `json:"stdout"`
		Stderr     string `json:"stderr"`
		ReturnCode int    `json:"return_code"`
	}{
		Stdout:     outBuf.String(),
		Stderr:     errBuf.String(),
		ReturnCode: exitCode,
	}

	curlOutputJSON, _ := json.Marshal(curlOutput)

	res.Header().Add("Content-Type", "application/json")
	res.Write(curlOutputJSON)
}
