package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"code.cloudfoundry.org/credhub-cli/credhub"
	"code.cloudfoundry.org/credhub-cli/credhub/auth"
	"code.cloudfoundry.org/credhub-cli/credhub/credentials/values"
	"code.cloudfoundry.org/credhub-cli/credhub/permissions"
	"code.cloudfoundry.org/credhub-cli/util"
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
)

func main() {
	var server Server

	server.Start()
}

type Server struct {
	sb *ServiceBroker
}

type bindRequest struct {
	AppGuid      string `json:"app_guid"`
	BindResource struct {
		CredentialClientId string `json:"credential_client_id"`
	} `json:"bind_resource"`
}

func (s *Server) Start() {
	router := mux.NewRouter()

	s.sb = &ServiceBroker{
		NameMap: make(map[string]string),
	}

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
	NameMap map[string]string
}

func WriteResponse(w http.ResponseWriter, code int, response string) {
	w.WriteHeader(code)
	fmt.Fprintf(w, string(response))
}

func (s *ServiceBroker) Catalog(w http.ResponseWriter, r *http.Request) {
	serviceUUID := uuid.NewV4().String()
	planUUID := uuid.NewV4().String()

	serviceName := "credhub-read"
	if os.Getenv("SERVICE_NAME") != "" {
		serviceName = os.Getenv("SERVICE_NAME")
	}

	catalog := `{
	"services": [{
		"name": "` + serviceName + `",
		"id": "` + serviceUUID + `",
		"description": "credhub read service for tests",
		"bindable": true,
		"plans": [{
			"name": "credhub-read-plan",
			"id": "` + planUUID + `",
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
	ch, err := credhub.New(
		util.AddDefaultSchemeIfNecessary(os.Getenv("CREDHUB_API")),
		credhub.SkipTLSValidation(true),
		credhub.Auth(auth.UaaClientCredentials(os.Getenv("CREDHUB_CLIENT"), os.Getenv("CREDHUB_SECRET"))),
	)

	if err != nil {
		fmt.Println("credhub client configuration failed: " + err.Error())
	}

	body := bindRequest{}
	err = json.NewDecoder(r.Body).Decode(&body)

	if err != nil {
		fmt.Println(err)
	}

	name := strconv.FormatInt(time.Now().UnixNano(), 10)
	storedJson := values.JSON{}
	storedJson["user-name"] = "pinkyPie"
	storedJson["password"] = "rainbowDash"

	cred, err := ch.SetJSON(name, storedJson)
	handleError(err)

	pathVariables := mux.Vars(r)
	s.NameMap[pathVariables["service_binding_guid"]] = name

	if body.AppGuid != "" {
		_, err = ch.AddPermissions(cred.Name, []permissions.Permission{{
			Actor:      "mtls-app:" + body.AppGuid,
			Operations: []string{"read"},
			Path: cred.Name,
		}})
		handleError(err)
	}

	credentials := `{
  "credentials": {
    "credhub-ref": "` + cred.Name + `"
  }
}`

	WriteResponse(w, http.StatusCreated, credentials)
}

func (s *ServiceBroker) UnBind(w http.ResponseWriter, r *http.Request) {
	ch, err := credhub.New(
		util.AddDefaultSchemeIfNecessary(os.Getenv("CREDHUB_API")),
		credhub.SkipTLSValidation(true),
		credhub.Auth(auth.UaaClientCredentials(os.Getenv("CREDHUB_CLIENT"), os.Getenv("CREDHUB_SECRET"))),
	)

	if err != nil {
		fmt.Println("credhub client configuration failed: " + err.Error())
	}

	pathVariables := mux.Vars(r)
	name := s.NameMap[pathVariables["service_binding_guid"]]

	err = ch.Delete(name)

	handleError(err)

	WriteResponse(w, http.StatusOK, "{}")
}

func handleError(err error) {
	if err != nil {
		fmt.Println(err)
		log.Fatal("Fatal", err)
	}
}
