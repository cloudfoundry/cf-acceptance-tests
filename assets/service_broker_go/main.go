package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Default configuration (equivalent to data.json in the Ruby broker)
// ---------------------------------------------------------------------------

const defaultDataJSON = `{
  "behaviors": {
    "catalog": {
      "sleep_seconds": 0,
      "status": 200,
      "body": {
        "services": [
          {
            "name": "fake-service",
            "id": "f479b64b-7c25-42e6-8d8f-e6d22c456c9b",
            "description": "fake service",
            "tags": ["no-sql", "relational"],
            "requires": ["route_forwarding"],
            "max_db_per_node": 5,
            "instances_retrievable": true,
            "bindings_retrievable": true,
            "bindable": true,
            "metadata": {
              "provider": {"name": "The name"},
              "listing": {
                "imageUrl": "http://catgifpage.com/cat.gif",
                "blurb": "fake broker that is fake",
                "longDescription": "A long time ago, in a galaxy far far away..."
              },
              "displayName": "The Fake Broker",
              "shareable": true
            },
            "dashboard_client": {
              "id": "sso-test",
              "secret": "sso-secret",
              "redirect_uri": "http://localhost:5551"
            },
            "plan_updateable": true,
            "plans": [
              {
                "name": "fake-plan",
                "id": "fake-plan-guid",
                "description": "Shared fake Server, 5tb persistent disk, 40 max concurrent connections",
                "max_storage_tb": 5,
                "metadata": {
                  "cost": 0,
                  "bullets": [
                    {"content": "Shared fake server"},
                    {"content": "5 TB storage"},
                    {"content": "40 concurrent connections"}
                  ]
                }
              },
              {
                "name": "fake-async-plan",
                "id": "fake-async-plan-guid",
                "description": "Shared fake Server, 5tb persistent disk, 40 max concurrent connections. 100 async",
                "max_storage_tb": 5,
                "metadata": {
                  "cost": 0,
                  "bullets": [{"content": "40 concurrent connections"}]
                }
              },
              {
                "name": "fake-async-only-plan",
                "id": "fake-async-only-plan-guid",
                "description": "Shared fake Server, 5tb persistent disk, 40 max concurrent connections. 100 async",
                "max_storage_tb": 5,
                "metadata": {
                  "cost": 0,
                  "bullets": [{"content": "40 concurrent connections"}]
                }
              }
            ]
          }
        ]
      }
    },
    "provision": {
      "fake-async-plan-guid": {
        "sleep_seconds": 0,
        "status": 202,
        "body": {}
      },
      "fake-async-only-plan-guid": {
        "async_only": true,
        "sleep_seconds": 0,
        "status": 202,
        "body": {}
      },
      "default": {
        "sleep_seconds": 0,
        "status": 200,
        "body": {}
      }
    },
    "fetch": {
      "default": {
        "in_progress": {
          "sleep_seconds": 0,
          "status": 200,
          "body": {"state": "in progress"}
        },
        "finished": {
          "sleep_seconds": 0,
          "status": 200,
          "body": {"state": "succeeded"}
        }
      }
    },
    "update": {
      "fake-async-plan-guid": {
        "sleep_seconds": 0,
        "status": 202,
        "body": {}
      },
      "fake-async-only-plan-guid": {
        "async_only": true,
        "sleep_seconds": 0,
        "status": 202,
        "body": {}
      },
      "default": {
        "sleep_seconds": 0,
        "status": 200,
        "body": {}
      }
    },
    "deprovision": {
      "fake-async-plan-guid": {
        "sleep_seconds": 0,
        "status": 202,
        "body": {}
      },
      "fake-async-only-plan-guid": {
        "async_only": true,
        "sleep_seconds": 0,
        "status": 202,
        "body": {}
      },
      "default": {
        "sleep_seconds": 0,
        "status": 200,
        "body": {}
      }
    },
    "bind": {
      "default": {
        "sleep_seconds": 0,
        "status": 201,
        "body": {
          "route_service_url": "https://logging-route-service.bosh-lite.env.wg-ard.ci.cloudfoundry.org",
          "credentials": {
            "uri": "fake-service://fake-user:fake-password@fake-host:3306/fake-dbname",
            "username": "fake-user",
            "password": "fake-password",
            "host": "fake-host",
            "port": 3306,
            "database": "fake-dbname"
          }
        }
      }
    },
    "unbind": {
      "default": {
        "sleep_seconds": 0,
        "status": 200,
        "body": {}
      }
    }
  },
  "service_instances": {},
  "service_bindings": {},
  "max_fetch_service_instance_requests": 1
}`

// ---------------------------------------------------------------------------
// Data types
// ---------------------------------------------------------------------------

// Behavior represents a single response behavior definition.
type Behavior struct {
	SleepSeconds float64                `json:"sleep_seconds"`
	Status       int                    `json:"status"`
	Body         map[string]interface{} `json:"body"`
	RawBody      *string                `json:"raw_body,omitempty"`
	AsyncOnly    bool                   `json:"async_only"`
}

// ServiceInstance holds the per-instance state.
type ServiceInstance struct {
	ProvisionData map[string]interface{} `json:"provision_data"`
	FetchCount    int                    `json:"fetch_count"`
	Deleted       bool                   `json:"deleted"`
}

func (si *ServiceInstance) PlanID() string {
	if si.ProvisionData == nil {
		return ""
	}
	if pid, ok := si.ProvisionData["plan_id"].(string); ok {
		return pid
	}
	return ""
}

func (si *ServiceInstance) Update(data map[string]interface{}) {
	for k, v := range data {
		si.ProvisionData[k] = v
	}
	si.FetchCount = 0
}

func (si *ServiceInstance) Delete() {
	si.Deleted = true
	si.FetchCount = 0
}

// ServiceBinding stores per-binding state.
type ServiceBinding struct {
	BindingData map[string]interface{} `json:"binding_data"`
	InstanceID  string                 `json:"instance_id"`
}

// DataSource is the in-memory broker state (equivalent to Ruby's DataSource).
type DataSource struct {
	// Raw behaviours as generic maps so we can deep-merge them.
	Behaviors                      map[string]interface{} `json:"behaviors"`
	ServiceInstancesRaw            map[string]interface{} `json:"service_instances"`
	ServiceBindingsRaw             map[string]interface{} `json:"service_bindings"`
	MaxFetchServiceInstanceRequests int                   `json:"max_fetch_service_instance_requests"`

	// Typed instances/bindings (populated alongside the raw maps)
	instances map[string]*ServiceInstance
	bindings  map[string]*ServiceBinding
}

func newDataSource() *DataSource {
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(defaultDataJSON), &raw); err != nil {
		log.Fatalf("failed to parse default config: %v", err)
	}
	ds := &DataSource{
		instances: make(map[string]*ServiceInstance),
		bindings:  make(map[string]*ServiceBinding),
	}
	ds.loadFromRaw(raw)
	return ds
}

func (ds *DataSource) loadFromRaw(raw map[string]interface{}) {
	if b, ok := raw["behaviors"].(map[string]interface{}); ok {
		ds.Behaviors = b
	}
	if ds.Behaviors == nil {
		ds.Behaviors = make(map[string]interface{})
	}
	if v, ok := raw["max_fetch_service_instance_requests"].(float64); ok {
		ds.MaxFetchServiceInstanceRequests = int(v)
	} else {
		ds.MaxFetchServiceInstanceRequests = 1
	}
	// service_instances and service_bindings are kept in typed maps
	if siRaw, ok := raw["service_instances"].(map[string]interface{}); ok {
		ds.ServiceInstancesRaw = siRaw
		// rebuild typed instances from raw (used by /config reset)
		for id, v := range siRaw {
			if m, ok := v.(map[string]interface{}); ok {
				si := &ServiceInstance{}
				if pd, ok := m["provision_data"].(map[string]interface{}); ok {
					si.ProvisionData = pd
				}
				if fc, ok := m["fetch_count"].(float64); ok {
					si.FetchCount = int(fc)
				}
				if del, ok := m["deleted"].(bool); ok {
					si.Deleted = del
				}
				ds.instances[id] = si
			}
		}
	} else {
		ds.ServiceInstancesRaw = make(map[string]interface{})
	}
	if sbRaw, ok := raw["service_bindings"].(map[string]interface{}); ok {
		ds.ServiceBindingsRaw = sbRaw
	} else {
		ds.ServiceBindingsRaw = make(map[string]interface{})
	}
}

// BehaviorForType returns the behavior map for a given type and plan_id.
// For catalog it returns the catalog behavior map directly.
// For other types it resolves plan_id -> default.
func (ds *DataSource) BehaviorForType(btype, planID string) (map[string]interface{}, error) {
	raw, ok := ds.Behaviors[btype]
	if !ok {
		return nil, fmt.Errorf("behavior object is missing key: %s", btype)
	}

	if btype == "catalog" {
		m, ok := raw.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("catalog behavior is not a map")
		}
		return m, nil
	}

	plans, ok := raw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("behavior for %s is not a map", btype)
	}

	if planID != "" {
		if v, ok := plans[planID]; ok {
			if m, ok := v.(map[string]interface{}); ok {
				return m, nil
			}
		}
	}

	if def, ok := plans["default"]; ok {
		if m, ok := def.(map[string]interface{}); ok {
			return m, nil
		}
	}

	return nil, fmt.Errorf("behavior for %s is missing response for plan_id %q and default response", btype, planID)
}

// parseBehavior converts a raw behavior map to a typed Behavior.
func parseBehavior(m map[string]interface{}) Behavior {
	b := Behavior{}
	if v, ok := m["sleep_seconds"].(float64); ok {
		b.SleepSeconds = v
	}
	if v, ok := m["status"].(float64); ok {
		b.Status = int(v)
	}
	if v, ok := m["body"].(map[string]interface{}); ok {
		b.Body = v
	}
	if v, ok := m["raw_body"].(string); ok {
		b.RawBody = &v
	}
	if v, ok := m["async_only"].(bool); ok {
		b.AsyncOnly = v
	}
	return b
}

// deepMerge merges src into dst following Ruby's merge! semantics:
// for each key in src, if dst[key] is a map and src[key] is a map, recurse;
// otherwise replace.
func deepMerge(dst, src map[string]interface{}) {
	for k, srcVal := range src {
		if dstMap, ok := dst[k].(map[string]interface{}); ok {
			if srcMap, ok := srcVal.(map[string]interface{}); ok {
				deepMerge(dstMap, srcMap)
				continue
			}
		}
		dst[k] = srcVal
	}
}

// WithoutInstancesOrBindings returns the config without instance/binding data.
func (ds *DataSource) WithoutInstancesOrBindings() map[string]interface{} {
	return map[string]interface{}{
		"behaviors":                          ds.Behaviors,
		"max_fetch_service_instance_requests": ds.MaxFetchServiceInstanceRequests,
	}
}

// AllData returns the full config including instances/bindings.
func (ds *DataSource) AllData() map[string]interface{} {
	instances := make(map[string]interface{})
	for id, si := range ds.instances {
		instances[id] = map[string]interface{}{
			"provision_data": si.ProvisionData,
			"fetch_count":    si.FetchCount,
			"deleted":        si.Deleted,
		}
	}
	bindings := make(map[string]interface{})
	for id, sb := range ds.bindings {
		bindings[id] = map[string]interface{}{
			"binding_data": sb.BindingData,
			"instance_id":  sb.InstanceID,
		}
	}
	return map[string]interface{}{
		"behaviors":                          ds.Behaviors,
		"service_instances":                  instances,
		"service_bindings":                   bindings,
		"max_fetch_service_instance_requests": ds.MaxFetchServiceInstanceRequests,
	}
}

// Merge deep-merges incoming config into the data source (POST /config semantics).
func (ds *DataSource) Merge(incoming map[string]interface{}) {
	// Handle service_instances specially — convert to typed structs
	if siRaw, ok := incoming["service_instances"].(map[string]interface{}); ok {
		for guid, v := range siRaw {
			if m, ok := v.(map[string]interface{}); ok {
				si := &ServiceInstance{}
				if pd, ok := m["provision_data"].(map[string]interface{}); ok {
					si.ProvisionData = pd
				} else {
					si.ProvisionData = make(map[string]interface{})
				}
				if fc, ok := m["fetch_count"].(float64); ok {
					si.FetchCount = int(fc)
				}
				if del, ok := m["deleted"].(bool); ok {
					si.Deleted = del
				}
				ds.instances[guid] = si
			}
		}
		// Remove from incoming before generic merge so we don't overwrite behaviors
		incoming = shallowCopyExcept(incoming, "service_instances", "service_bindings")
	}

	// Now deep-merge remaining keys
	for key, val := range incoming {
		if key == "service_instances" || key == "service_bindings" {
			continue
		}
		// Top-level key merge: if existing value is a map and incoming is a map, deep merge
		switch key {
		case "behaviors":
			if srcMap, ok := val.(map[string]interface{}); ok {
				deepMerge(ds.Behaviors, srcMap)
			} else {
				ds.Behaviors = make(map[string]interface{})
			}
		case "max_fetch_service_instance_requests":
			if v, ok := val.(float64); ok {
				ds.MaxFetchServiceInstanceRequests = int(v)
			}
		default:
			// For any other key: if existing is a map and new is a map, deep merge
			// (we don't have other top-level map keys in the current schema, but be safe)
		}
	}
}

func shallowCopyExcept(m map[string]interface{}, excludeKeys ...string) map[string]interface{} {
	excluded := make(map[string]bool)
	for _, k := range excludeKeys {
		excluded[k] = true
	}
	out := make(map[string]interface{})
	for k, v := range m {
		if !excluded[k] {
			out[k] = v
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Server
// ---------------------------------------------------------------------------

type server struct {
	mu               sync.RWMutex
	ds               *DataSource
	cfApiInfoLocation string
}

func newServer() *server {
	return &server{
		ds: newDataSource(),
	}
}

func (s *server) logRequest(r *http.Request, body string) {
	log.Printf("REQUEST: %s %s %s", r.Method, r.URL.Path, r.URL.RawQuery)
	log.Printf("REQUEST BODY: %s", body)
}

func (s *server) writeJSON(w http.ResponseWriter, status int, v interface{}) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(data)
	log.Printf("RESPONSE: status=%d body=%s", status, string(data))
}

func (s *server) writeRaw(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	fmt.Fprint(w, body)
	log.Printf("RESPONSE: status=%d body=%s", status, body)
}

func (s *server) respondWithBehavior(w http.ResponseWriter, behavior Behavior, acceptsIncomplete bool) {
	if behavior.SleepSeconds > 0 {
		time.Sleep(time.Duration(behavior.SleepSeconds * float64(time.Second)))
	}

	if behavior.AsyncOnly && !acceptsIncomplete {
		s.writeJSON(w, 422, map[string]interface{}{
			"error":       "AsyncRequired",
			"description": "This service plan requires client support for asynchronous service operations.",
		})
		return
	}

	if behavior.RawBody != nil {
		s.writeRaw(w, behavior.Status, *behavior.RawBody)
		return
	}
	s.writeJSON(w, behavior.Status, behavior.Body)
}

func acceptsIncomplete(r *http.Request) bool {
	return r.URL.Query().Get("accepts_incomplete") == "true"
}

func readBody(r *http.Request) ([]byte, error) {
	defer r.Body.Close()
	return io.ReadAll(r.Body)
}

func parseJSONBody(r *http.Request) (map[string]interface{}, []byte, error) {
	data, err := readBody(r)
	if err != nil {
		return nil, nil, err
	}
	var m map[string]interface{}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &m); err != nil {
			return nil, data, err
		}
	}
	if m == nil {
		m = make(map[string]interface{})
	}
	return m, data, nil
}

// captureApiInfoLocation extracts X-Api-Info-Location from /v2/ requests.
func (s *server) captureApiInfoLocation(r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/v2/") {
		if v := r.Header.Get("X-Api-Info-Location"); v != "" {
			s.mu.Lock()
			s.cfApiInfoLocation = v
			s.mu.Unlock()
		}
	}
}

// ---------------------------------------------------------------------------
// Route handlers
// ---------------------------------------------------------------------------

func (s *server) handleCatalog(w http.ResponseWriter, r *http.Request) {
	s.captureApiInfoLocation(r)
	s.mu.RLock()
	bm, err := s.ds.BehaviorForType("catalog", "")
	s.mu.RUnlock()
	if err != nil {
		s.writeJSON(w, 500, map[string]interface{}{"error": err.Error()})
		return
	}
	b := parseBehavior(bm)
	s.respondWithBehavior(w, b, false)
}

func (s *server) handleProvision(w http.ResponseWriter, r *http.Request, id string) {
	s.captureApiInfoLocation(r)
	jsonBody, raw, err := parseJSONBody(r)
	s.logRequest(r, string(raw))
	if err != nil {
		s.writeJSON(w, 400, map[string]interface{}{"error": err.Error()})
		return
	}

	s.mu.Lock()
	si := &ServiceInstance{
		ProvisionData: jsonBody,
		FetchCount:    0,
	}
	s.ds.instances[id] = si
	planID := si.PlanID()
	bm, berr := s.ds.BehaviorForType("provision", planID)
	s.mu.Unlock()

	if berr != nil {
		s.writeJSON(w, 500, map[string]interface{}{"error": berr.Error()})
		return
	}
	b := parseBehavior(bm)
	s.respondWithBehavior(w, b, acceptsIncomplete(r))
}

func (s *server) handleLastOperation(w http.ResponseWriter, r *http.Request, id string) {
	s.captureApiInfoLocation(r)
	s.logRequest(r, "")

	s.mu.Lock()
	si, ok := s.ds.instances[id]
	if !ok {
		s.mu.Unlock()
		s.writeJSON(w, 200, map[string]interface{}{
			"state":       "failed",
			"description": fmt.Sprintf("Broker could not find service instance by the given id %s", id),
		})
		return
	}

	si.FetchCount++
	fetchCount := si.FetchCount
	maxFetch := s.ds.MaxFetchServiceInstanceRequests
	planID := si.PlanID()

	fetchBehaviors, berr := s.ds.BehaviorForType("fetch", planID)
	s.mu.Unlock()

	if berr != nil {
		s.writeJSON(w, 500, map[string]interface{}{"error": berr.Error()})
		return
	}

	var state string
	if fetchCount > maxFetch {
		state = "finished"
	} else {
		state = "in_progress"
	}

	stateBehaviorRaw, ok := fetchBehaviors[state]
	if !ok {
		s.writeJSON(w, 500, map[string]interface{}{"error": fmt.Sprintf("fetch behavior missing state %q", state)})
		return
	}
	stateBehaviorMap, ok := stateBehaviorRaw.(map[string]interface{})
	if !ok {
		s.writeJSON(w, 500, map[string]interface{}{"error": "fetch state behavior is not a map"})
		return
	}

	b := parseBehavior(stateBehaviorMap)
	if b.SleepSeconds > 0 {
		time.Sleep(time.Duration(b.SleepSeconds * float64(time.Second)))
	}
	if b.RawBody != nil {
		s.writeRaw(w, b.Status, *b.RawBody)
		return
	}
	s.writeJSON(w, b.Status, b.Body)
}

func (s *server) handleBindingLastOperation(w http.ResponseWriter, r *http.Request) {
	s.captureApiInfoLocation(r)
	s.logRequest(r, "")
	s.writeJSON(w, 200, map[string]interface{}{
		"state":       "succeeded",
		"description": "100%",
	})
}

func (s *server) handleUpdate(w http.ResponseWriter, r *http.Request, id string) {
	s.captureApiInfoLocation(r)
	jsonBody, raw, err := parseJSONBody(r)
	s.logRequest(r, string(raw))
	if err != nil {
		s.writeJSON(w, 400, map[string]interface{}{"error": err.Error()})
		return
	}

	planID, _ := jsonBody["plan_id"].(string)

	s.mu.Lock()
	bm, berr := s.ds.BehaviorForType("update", planID)
	if berr == nil {
		b := parseBehavior(bm)
		if b.Status == 200 || b.Status == 202 {
			if si, ok := s.ds.instances[id]; ok {
				si.Update(jsonBody)
			}
		}
	}
	s.mu.Unlock()

	if berr != nil {
		s.writeJSON(w, 500, map[string]interface{}{"error": berr.Error()})
		return
	}
	b := parseBehavior(bm)
	s.respondWithBehavior(w, b, acceptsIncomplete(r))
}

func (s *server) handleDeprovision(w http.ResponseWriter, r *http.Request, id string) {
	s.captureApiInfoLocation(r)
	s.logRequest(r, "")

	s.mu.Lock()
	si, hasSI := s.ds.instances[id]
	var planID string
	if hasSI {
		planID = si.PlanID()
		si.Delete()
	}
	bm, berr := s.ds.BehaviorForType("deprovision", planID)
	s.mu.Unlock()

	if berr != nil {
		s.writeJSON(w, 500, map[string]interface{}{"error": berr.Error()})
		return
	}
	b := parseBehavior(bm)
	s.respondWithBehavior(w, b, acceptsIncomplete(r))
}

func (s *server) handleBind(w http.ResponseWriter, r *http.Request, instanceID, bindingID string) {
	s.captureApiInfoLocation(r)
	jsonBody, raw, err := parseJSONBody(r)
	s.logRequest(r, string(raw))
	if err != nil {
		s.writeJSON(w, 400, map[string]interface{}{"error": err.Error()})
		return
	}

	planID, _ := jsonBody["plan_id"].(string)

	s.mu.Lock()
	s.ds.bindings[bindingID] = &ServiceBinding{
		BindingData: jsonBody,
		InstanceID:  instanceID,
	}
	bm, berr := s.ds.BehaviorForType("bind", planID)
	s.mu.Unlock()

	if berr != nil {
		s.writeJSON(w, 500, map[string]interface{}{"error": berr.Error()})
		return
	}
	b := parseBehavior(bm)
	s.respondWithBehavior(w, b, acceptsIncomplete(r))
}

func (s *server) handleUnbind(w http.ResponseWriter, r *http.Request, instanceID, bindingID string) {
	s.captureApiInfoLocation(r)
	s.logRequest(r, "")

	s.mu.Lock()
	sb, hasSB := s.ds.bindings[bindingID]
	var planID string
	if hasSB {
		planID, _ = sb.BindingData["plan_id"].(string)
		delete(s.ds.bindings, bindingID)
	}
	bm, berr := s.ds.BehaviorForType("unbind", planID)
	s.mu.Unlock()

	if berr != nil {
		s.writeJSON(w, 500, map[string]interface{}{"error": berr.Error()})
		return
	}
	b := parseBehavior(bm)
	s.respondWithBehavior(w, b, acceptsIncomplete(r))
}

func (s *server) handleGetInstance(w http.ResponseWriter, r *http.Request, instanceID string) {
	s.captureApiInfoLocation(r)
	s.logRequest(r, "")

	s.mu.RLock()
	si, ok := s.ds.instances[instanceID]
	s.mu.RUnlock()

	if !ok {
		s.writeJSON(w, 404, map[string]interface{}{"description": "not found"})
		return
	}
	s.writeJSON(w, 200, si.ProvisionData)
}

func (s *server) handleFetchBinding(w http.ResponseWriter, r *http.Request, instanceID, bindingID string) {
	s.captureApiInfoLocation(r)
	s.logRequest(r, "")

	s.mu.Lock()
	sb, ok := s.ds.bindings[bindingID]
	if !ok {
		s.mu.Unlock()
		s.writeJSON(w, 404, map[string]interface{}{"description": "binding not found"})
		return
	}
	planID, _ := sb.BindingData["plan_id"].(string)
	bm, berr := s.ds.BehaviorForType("fetch_service_binding", planID)
	s.mu.Unlock()

	if berr != nil {
		s.writeJSON(w, 500, map[string]interface{}{"error": berr.Error()})
		return
	}

	b := parseBehavior(bm)

	// Ruby: response_body['body'].merge!(binding_data)
	// i.e. the stored binding data is merged INTO the body
	if b.Body == nil {
		b.Body = make(map[string]interface{})
	}
	s.mu.RLock()
	for k, v := range sb.BindingData {
		b.Body[k] = v
	}
	s.mu.RUnlock()

	s.respondWithBehavior(w, b, false)
}

func (s *server) handleConfigGet(w http.ResponseWriter, r *http.Request) {
	s.logRequest(r, "")
	s.mu.RLock()
	data := s.ds.WithoutInstancesOrBindings()
	s.mu.RUnlock()
	s.writeJSON(w, 200, data)
}

func (s *server) handleConfigGetAll(w http.ResponseWriter, r *http.Request) {
	s.logRequest(r, "")
	s.mu.RLock()
	data := s.ds.AllData()
	s.mu.RUnlock()
	s.writeJSON(w, 200, data)
}

func (s *server) handleConfigPost(w http.ResponseWriter, r *http.Request) {
	jsonBody, raw, err := parseJSONBody(r)
	s.logRequest(r, string(raw))
	if err != nil {
		s.writeJSON(w, 400, map[string]interface{}{"error": err.Error()})
		return
	}

	s.mu.Lock()
	s.ds.Merge(jsonBody)
	data := s.ds.WithoutInstancesOrBindings()
	s.mu.Unlock()

	s.writeJSON(w, 200, data)
}

func (s *server) handleConfigReset(w http.ResponseWriter, r *http.Request) {
	s.logRequest(r, "")
	s.mu.Lock()
	s.ds = newDataSource()
	data := s.ds.WithoutInstancesOrBindings()
	s.mu.Unlock()
	s.writeJSON(w, 200, data)
}

func (s *server) handleCfApiInfoURL(w http.ResponseWriter, r *http.Request) {
	s.logRequest(r, "")
	s.mu.RLock()
	loc := s.cfApiInfoLocation
	s.mu.RUnlock()

	if loc == "" {
		s.writeJSON(w, 503, map[string]interface{}{
			"error":   true,
			"message": "CF API info URL not known - either the cloud controller has not called the broker API yet, or it has failed to include a X-Api-Info-Location header that was a valid URL",
			"path":    r.URL.String(),
			"type":    "503",
		})
		return
	}
	w.WriteHeader(200)
	fmt.Fprint(w, loc)
	log.Printf("RESPONSE: status=200 body=%s", loc)
}

// ---------------------------------------------------------------------------
// Router
// ---------------------------------------------------------------------------

// trimTrailingSlash normalises a path by removing a trailing slash except for "/".
func trimTrailingSlash(p string) string {
	if len(p) > 1 && strings.HasSuffix(p, "/") {
		return p[:len(p)-1]
	}
	return p
}

// splitPath splits the URL path into clean segments.
func splitPath(p string) []string {
	p = trimTrailingSlash(p)
	parts := strings.Split(p, "/")
	// Remove empty first element from leading "/"
	if len(parts) > 0 && parts[0] == "" {
		parts = parts[1:]
	}
	return parts
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	parts := splitPath(r.URL.Path)
	method := r.Method

	// /v2/catalog
	if len(parts) == 2 && parts[0] == "v2" && parts[1] == "catalog" && method == http.MethodGet {
		s.handleCatalog(w, r)
		return
	}

	// /v2/service_instances/...
	if len(parts) >= 3 && parts[0] == "v2" && parts[1] == "service_instances" {
		instanceID := parts[2]

		// /v2/service_instances/:id  (GET, PUT, PATCH, DELETE)
		if len(parts) == 3 {
			switch method {
			case http.MethodPut:
				s.handleProvision(w, r, instanceID)
			case http.MethodPatch:
				s.handleUpdate(w, r, instanceID)
			case http.MethodDelete:
				s.handleDeprovision(w, r, instanceID)
			case http.MethodGet:
				s.handleGetInstance(w, r, instanceID)
			default:
				http.NotFound(w, r)
			}
			return
		}

		// /v2/service_instances/:id/last_operation
		if len(parts) == 4 && parts[3] == "last_operation" && method == http.MethodGet {
			s.handleLastOperation(w, r, instanceID)
			return
		}

		// /v2/service_instances/:instance_id/service_bindings/...
		if len(parts) >= 5 && parts[3] == "service_bindings" {
			bindingID := parts[4]

			// /v2/service_instances/:instance_id/service_bindings/:id  (PUT, DELETE, GET)
			if len(parts) == 5 {
				switch method {
				case http.MethodPut:
					s.handleBind(w, r, instanceID, bindingID)
				case http.MethodDelete:
					s.handleUnbind(w, r, instanceID, bindingID)
				case http.MethodGet:
					s.handleFetchBinding(w, r, instanceID, bindingID)
				default:
					http.NotFound(w, r)
				}
				return
			}

			// /v2/service_instances/:instance_id/service_bindings/:binding_id/last_operation
			if len(parts) == 6 && parts[5] == "last_operation" && method == http.MethodGet {
				s.handleBindingLastOperation(w, r)
				return
			}
		}
	}

	// /config routes
	if len(parts) >= 1 && parts[0] == "config" {
		if len(parts) == 1 {
			switch method {
			case http.MethodGet:
				s.handleConfigGet(w, r)
			case http.MethodPost:
				s.handleConfigPost(w, r)
			default:
				http.NotFound(w, r)
			}
			return
		}
		if len(parts) == 2 {
			switch parts[1] {
			case "all":
				if method == http.MethodGet {
					s.handleConfigGetAll(w, r)
					return
				}
			case "reset":
				if method == http.MethodPost {
					s.handleConfigReset(w, r)
					return
				}
			}
		}
	}

	// /cf_api_info_url
	if len(parts) == 1 && parts[0] == "cf_api_info_url" && method == http.MethodGet {
		s.handleCfApiInfoURL(w, r)
		return
	}

	http.NotFound(w, r)
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := newServer()

	httpServer := &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%s", port),
		Handler: srv,
	}

	log.Printf("Service broker listening on :%s", port)
	if err := httpServer.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
