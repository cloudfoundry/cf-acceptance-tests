package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"code.cloudfoundry.org/credhub-cli/credhub"
	"code.cloudfoundry.org/credhub-cli/credhub/auth"
	"code.cloudfoundry.org/credhub-cli/util"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	// Create a new CredHub client
	ch, err := credhub.New(
		util.AddDefaultSchemeIfNecessary(os.Getenv("CREDHUB_API")),
		credhub.SkipTLSValidation(true),
		credhub.Auth(auth.UaaClientCredentials(os.Getenv("CREDHUB_CLIENT"), os.Getenv("CREDHUB_SECRET"))),
	)
	if err != nil {
		log.Fatal("Failed to create CredHub client: ", err)
	}

	// Create a map of service binding GUIDs to track the registered service instances
	bindings := make(map[string]string)

	// Create a router and register the service broker handlers
	router := chi.NewRouter()
	router.Use(middleware.Recoverer)

	router.Get("/v2/catalog", catalogHandler)
	router.Route("/v2/service_instances", func(r chi.Router) {
		r.Put("/{service_instance_guid}", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("{}"))
		})
		r.Delete("/{service_instance_guid}", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("{}"))
		})
		r.Route("/{service_instance_guid}/service_bindings", func(r chi.Router) {
			r.Put("/{service_binding_guid}", bindHandler(ch, bindings))
			r.Delete("/{service_binding_guid}", unBindHandler(ch, bindings))
		})
	})

	// Retrieve the port to listen on from the environment
	port, err := strconv.Atoi(os.Getenv("PORT"))
	if err != nil {
		log.Fatal("Failed to parse PORT: ", err)
	}

	// Start the HTTP server
	log.Printf("Server starting, listening on port %d...", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), router))
}
