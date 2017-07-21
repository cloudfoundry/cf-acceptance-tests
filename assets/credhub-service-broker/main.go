package main

import (
	"net/http"
	"os"
	"fmt"
	"github.com/gorilla/mux"
	"encoding/json"
	"bytes"
	"io/ioutil"
	"log"
	"crypto/tls"
	"time"
	"strconv"
)

func main() {
	var server Server

	server.Start()
}

type Server struct {
	sb *ServiceBroker
}

type bindRequest struct {
	AppGuid string `json:"app_guid"`
}

type permissions struct {
	Actor string `json:"actor"`
	Operations []string `json:"operations"`
}

func (s *Server) Start() {
	router := mux.NewRouter()

	router.HandleFunc("/v2/catalog", s.sb.Catalog).Methods("GET")
	router.HandleFunc("/v2/service_instances/{service_instance_guid}", s.sb.CreateServiceInstance).Methods("PUT")
	router.HandleFunc("/v2/service_instances/{service_instance_guid}", s.sb.RemoveServiceInstance).Methods("DELETE")
	router.HandleFunc("/v2/service_instances/{service_instance_guid}/service_bindings/{service_binding_guid}", s.sb.Bind).Methods("PUT")
	router.HandleFunc("/v2/service_instances/{service_instance_guid}/service_bindings/{service_binding_guid}", s.sb.UnBind).Methods("DELETE")

	http.Handle("/", router)

	cfPort := os.Getenv("PORT")

	fmt.Println("Server started, listening on port " + cfPort + "...")
	fmt.Println("CTL-C to break out of broker")
	fmt.Println(http.ListenAndServe(":"+cfPort, nil))
}

type ServiceBroker struct {

}

func WriteResponse(w http.ResponseWriter, code int, response string) {
	w.WriteHeader(code)
	fmt.Fprintf(w, string(response))
}

func (s *ServiceBroker) Catalog(w http.ResponseWriter, r *http.Request) {
	catalog := `{
	"services": [{
		"name": "credhub-read",
		"id": "110049a1-3e3e-4ab4-84c3-f41c430ad1f9",
		"description": "credhub read service for tests",
		"bindable": true,
		"plans": [{
			"name": "credhub-read-plan",
			"id": "12873401-28fb-44aa-b931-7f09806ca76f",
			"description": "credhub read service for tests"
		}]
	}]
}`

	WriteResponse(w, http.StatusOK, catalog)
}

func (s *ServiceBroker) CreateServiceInstance(w http.ResponseWriter, r *http.Request) {
	WriteResponse(w, http.StatusCreated, "{}")
}

func (s *ServiceBroker) RemoveServiceInstance(w http.ResponseWriter, r *http.Request) {
	WriteResponse(w, http.StatusOK, "{}")
}

func (s *ServiceBroker) Bind(w http.ResponseWriter, r *http.Request) {
	body := bindRequest{}
	err := json.NewDecoder(r.Body).Decode(&body)

	fmt.Println("parsed request body: ", body)

	if err != nil {
		fmt.Println("OH NO: ", err)
	}

	storedJson := map[string]string {
		"user-name": "pinkyPie",
		"password": "rainbowDash",
	}

	permissionJson := permissions{
		Actor: "mtls-app:" + body.AppGuid,
		Operations: []string{"read", "delete"},
	}

	putData := map[string]interface{}{
		"name":  strconv.FormatInt(time.Now().UnixNano(), 10),
		"type": "json",
		"value": storedJson,
		"additional_permissions": []permissions{permissionJson},
	}

	result, err := mtlsPutRequest(
		os.Getenv("CREDHUB_API")+"/api/v1/data",
		putData)

	handleError(err)

	responseData := make(map[string]string)

	json.Unmarshal([]byte(result), &responseData)

	credentials := `{
  "credentials": {
    "credhub-ref": "`+ responseData["name"] + `"
  }
}`

	WriteResponse(w, http.StatusCreated, credentials)
}

func (s *ServiceBroker) UnBind(w http.ResponseWriter, r *http.Request) {
	WriteResponse(w, http.StatusOK, "{}")
}

func mtlsPutRequest(url string, postData map[string]interface{}) (string, error) {
	client, err := createMtlsClient()

	jsonValue, err := json.Marshal(postData)
	handleError(err)

	request, err := http.NewRequest("PUT", url,bytes.NewBuffer(jsonValue))
	request.Header.Set("Content-type", "application/json")

	handleError(err)

	resp, err := client.Do(request)

	handleError(err)

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func handleError(err error) {
	if err != nil {
		fmt.Println(err)
		log.Fatal("Fatal", err)
	}
}


func createMtlsClient() (*http.Client, error) {
	clientCertPath := os.Getenv("CF_INSTANCE_CERT")
	clientKeyPath := os.Getenv("CF_INSTANCE_KEY")

	_, err := os.Stat(clientCertPath)
	handleError(err)
	_, err = os.Stat(clientKeyPath)
	handleError(err)

	clientCertificate, err := tls.LoadX509KeyPair(clientCertPath, clientKeyPath)
	handleError(err)

	tlsConf := &tls.Config{
		Certificates: []tls.Certificate{clientCertificate},
	}

	transport := &http.Transport{TLSClientConfig: tlsConf}
	client := &http.Client{Transport: transport}

	return client, err
}
