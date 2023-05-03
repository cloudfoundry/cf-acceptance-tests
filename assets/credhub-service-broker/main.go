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
	"code.cloudfoundry.org/credhub-cli/util"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	uuid "github.com/satori/go.uuid"
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
	router := chi.NewRouter()
	router.Use(middleware.Recoverer)

	s.sb = &ServiceBroker{
		NameMap: make(map[string]string),
	}

	router.Get("/v2/catalog", s.sb.Catalog)

	router.Route("/v2/service_instances", func(r chi.Router) {
		r.Put("/{service_instance_guid}", s.sb.CreateServiceInstance)
		r.Delete("/{service_instance_guid}", s.sb.RemoveServiceInstance)

		r.Route("/{service_instance_guid}/service_bindings", func(r chi.Router) {
			r.Put("/{service_binding_guid}", s.sb.Bind)
			r.Delete("/{service_binding_guid}", s.sb.UnBind)
		})
	})

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
	fmt.Fprint(w, string(response))
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

	s.NameMap[chi.URLParam(r, "service_binding_guid")] = name

	if body.AppGuid != "" {
		_, err = ch.AddPermission(cred.Name, "mtls-app:"+body.AppGuid, []string{"read"})
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

	name := s.NameMap[chi.URLParam(r, "service_binding_guid")]

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
