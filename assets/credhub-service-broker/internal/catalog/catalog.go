package catalog

import (
	"encoding/json"
	"log"
	"net/http"
)

// Plan is a struct that represents a plan in the catalog.
type Plan struct {
	Name        string `json:"name"`
	ID          string `json:"id"`
	Description string `json:"description"`
}

// Service is a struct that represents a service in the catalog.
type Service struct {
	Name        string `json:"name"`
	ID          string `json:"id"`
	Description string `json:"description"`
	Bindable    bool   `json:"bindable"`
	Plans       []Plan `json:"plans"`
}

// Catalog is a struct that represents the catalog.
type Catalog struct {
	Services []Service `json:"services"`
}

var _ http.Handler = (*Catalog)(nil)

// New creates a new Catalog with the given service name, service ID, and plan ID.
func New(serviceName, serviceID, planID string) *Catalog {
	return &Catalog{
		Services: []Service{
			{
				Name:        serviceName,
				ID:          serviceID,
				Description: "credhub read service for tests",
				Bindable:    true,
				Plans: []Plan{
					{
						Name:        "credhub-read-plan",
						ID:          planID,
						Description: "credhub read plan for tests",
					},
				},
			},
		},
	}
}

// ServeHTTP serves the HTTP request by marshalling the catalog to JSON and writing it to the response writer.
func (c *Catalog) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	catalogJSON, err := json.Marshal(c)
	if err != nil {
		log.Println("Failed to marshal catalog: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(catalogJSON)
}
