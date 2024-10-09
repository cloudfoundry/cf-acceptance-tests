package main

import (
	"fmt"
	"log"
	"net/http"

	"code.cloudfoundry.org/credhub-cli/credhub"
	"code.cloudfoundry.org/credhub-cli/credhub/auth"
	"code.cloudfoundry.org/credhub-cli/util"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	// Load configuration
	cfg := LoadConfig()

	// Create a new CredHub client
	ch, err := credhub.New(
		util.AddDefaultSchemeIfNecessary(cfg.Credhub.API),
		credhub.SkipTLSValidation(true),
		credhub.Auth(auth.UaaClientCredentials(cfg.Credhub.Client, cfg.Credhub.Secret)),
	)
	if err != nil {
		log.Panic("Failed to create CredHub client: ", err)
	}

	// Create a map of service binding GUIDs to track the registered service instances
	bindings := make(map[string]string)

	// Create a router and register the service broker handlers
	router := chi.NewRouter()
	router.Use(middleware.Recoverer)

	router.Get("/v2/catalog", catalogHandler(cfg))
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

	// Start the HTTP server
	log.Printf("Server starting, listening on port %d...", cfg.Port)
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: router,
	}
	log.Fatal(server.ListenAndServe())
}
