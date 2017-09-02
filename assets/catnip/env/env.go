package env

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
)

// get '/env/:name' do
// ENV[params[:name]]
// end

// get '/env' do
// ENV.to_hash.to_s
// end

// get '/env.json' do
// ENV.to_hash.to_json
// end

func NameHandler(res http.ResponseWriter, req *http.Request) {
	io.WriteString(res, os.Getenv(mux.Vars(req)["name"]))
}

func JsonHandler(res http.ResponseWriter, req *http.Request) {
	envMap := make(map[string]string)
	for _, e := range os.Environ() {
		kv := strings.Split(e, "=")
		envMap[kv[0]] = kv[1]
	}

	envJSON, _ := json.Marshal(envMap)

	res.Header().Add("Content-Type", "application/json")
	res.Write(envJSON)
}
