package env

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
)

func NameHandler(res http.ResponseWriter, req *http.Request) {
	io.WriteString(res, os.Getenv(mux.Vars(req)["name"]))
}

func JSONHandler(res http.ResponseWriter, req *http.Request) {
	envMap := make(map[string]string)
	for _, e := range os.Environ() {
		kv := strings.Split(e, "=")
		envMap[kv[0]] = kv[1]
	}

	envJSON, _ := json.Marshal(envMap)

	res.Header().Add("Content-Type", "application/json")
	res.Write(envJSON)
}

func InstanceGuidHandler(res http.ResponseWriter, req *http.Request) {
	io.WriteString(res, InstanceGuid())
}

func InstanceGuid() string {
	return os.Getenv("CF_INSTANCE_GUID")
}
