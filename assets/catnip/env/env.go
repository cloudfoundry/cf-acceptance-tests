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

func InstanceIdHandler(res http.ResponseWriter, req *http.Request) {
	instanceId, _ := InstanceId()

	io.WriteString(res, instanceId)
}

func InstanceId() (string, error) {
	vcapString := os.Getenv("VCAP_APPLICATION")
	vcapMap := make(map[string]string)

	err := json.Unmarshal([]byte(vcapString), &vcapMap)
	if err != nil {
		return "", err
	}

	return vcapMap["instance_id"], nil
}
