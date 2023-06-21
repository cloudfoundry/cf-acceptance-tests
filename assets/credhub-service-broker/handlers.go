package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"code.cloudfoundry.org/credhub-cli/credhub"
	"code.cloudfoundry.org/credhub-cli/credhub/credentials/values"
	"github.com/go-chi/chi/v5"
	uuid "github.com/satori/go.uuid"
)

var (
	SERVICE_NAME string
	SERVICE_UUID string
	PLAN_UUID    string
)

func init() {
	SERVICE_NAME = os.Getenv("SERVICE_NAME")
	if SERVICE_NAME == "" {
		SERVICE_NAME = "credhub-read"
	}
	SERVICE_UUID = uuid.NewV4().String()
	PLAN_UUID = uuid.NewV4().String()
}

func catalogHandler(w http.ResponseWriter, r *http.Request) {
	// Create a new catalog response
	type Plans struct {
		Name        string `json:"name"`
		ID          string `json:"id"`
		Description string `json:"description"`
	}
	type Services struct {
		Name        string  `json:"name"`
		ID          string  `json:"id"`
		Description string  `json:"description"`
		Bindable    bool    `json:"bindable"`
		Plans       []Plans `json:"plans"`
	}
	catalog := struct {
		Services []Services `json:"services"`
	}{
		Services: []Services{
			{
				Name:        SERVICE_NAME,
				ID:          SERVICE_UUID,
				Description: "credhub read service for tests",
				Bindable:    true,
				Plans: []Plans{
					{
						Name:        "credhub-read-plan",
						ID:          PLAN_UUID,
						Description: "credhub read plan for tests",
					},
				},
			},
		},
	}

	// Marshal the catalog response to JSON
	catalogJSON, err := json.Marshal(catalog)
	if err != nil {
		log.Println("Failed to marshal catalog response: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Write the catalog response to the response writer
	w.WriteHeader(http.StatusOK)
	w.Write(catalogJSON) //nolint:errcheck
}

func bindHandler(ch *credhub.CredHub, bindings map[string]string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse URL parameters
		sbGUID := chi.URLParam(r, "service_binding_guid")
		if sbGUID == "" {
			log.Println("Missing service binding GUID")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Parse the request body
		var bindRequest struct {
			AppGuid string `json:"app_guid"`
		}
		err := json.NewDecoder(r.Body).Decode(&bindRequest)
		if err != nil {
			log.Println("Failed to parse bind request: ", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Set a credential in CredHub
		name := strconv.FormatInt(time.Now().UnixNano(), 10)
		value := values.JSON{
			"user-name": "pinkyPie",
			"password":  "rainbowDash",
		}
		cred, err := ch.SetJSON(name, value)
		if err != nil {
			log.Println("Failed to set credential: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Add the binding to the bindings map
		bindings[sbGUID] = cred.Name

		// Give app access to the credential, if AppGuid is provided
		if bindRequest.AppGuid != "" {
			_, err = ch.AddPermission(cred.Name, "mtls-app:"+bindRequest.AppGuid, []string{"read"})
			if err != nil {
				log.Println("Failed to add permission: ", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}

		// Create a new binding response
		type Credentials struct {
			CredHubRef string `json:"credhub-ref"`
		}
		binding := struct {
			Credentials Credentials `json:"credentials"`
		}{
			Credentials: Credentials{
				CredHubRef: cred.Name,
			},
		}

		// Marshal the binding response to JSON
		bindingJSON, err := json.Marshal(binding)
		if err != nil {
			log.Println("Failed to marshal binding response: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Write the binding response to the response writer
		w.WriteHeader(http.StatusCreated)
		w.Write(bindingJSON) //nolint:errcheck
	}
}

func unBindHandler(ch *credhub.CredHub, bindings map[string]string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse URL parameters
		sbGUID := chi.URLParam(r, "service_binding_guid")
		if sbGUID == "" {
			log.Println("Missing service binding GUID")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Get the credential name from the bindings map
		credentialName, ok := bindings[sbGUID]
		if !ok {
			log.Println("Failed to find credential name for service binding GUID: ", sbGUID)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Delete the credential from CredHub
		err := ch.Delete(credentialName)
		if err != nil {
			log.Println("Failed to delete credential: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Delete the binding from the bindings map
		delete(bindings, sbGUID)

		// Write an empty JSON object to the response writer
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}")) //nolint:errcheck
	}
}
