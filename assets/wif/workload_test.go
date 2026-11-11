package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWorkloadUsesExchangedCredentialAndRedactsBothTokens(t *testing.T) {
	const assertion = "uaa-secret-assertion"
	const credential = "sts-secret-credential"
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		if r.Form.Get("resource") != "urn:test:federation" {
			t.Error("missing resource")
		}
		json.NewEncoder(w).Encode(map[string]string{"access_token": assertion})
	})
	mux.HandleFunc("/exchange", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		if r.Form.Get("subject_token") != assertion {
			t.Error("wrong assertion")
		}
		json.NewEncoder(w).Encode(map[string]string{"access_token": credential})
	})
	mux.HandleFunc("/resource", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+credential {
			w.WriteHeader(401)
			return
		}
		json.NewEncoder(w).Encode(map[string]bool{"authorized": true})
	})
	mux.HandleFunc("/v3/apps", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(401) })
	s := httptest.NewServer(mux)
	defer s.Close()
	result := workload{Client: s.Client(), TokenEndpoint: s.URL + "/token", ClientID: "account", Resource: "urn:test:federation", STSURL: s.URL, CAPITarget: s.URL}.run("exchange")
	if result["exchange_status"] != 200 || result["resource_status"] != 200 || result["exchanged_capi_status"] != 401 {
		t.Fatal("flow did not complete")
	}
	encoded, _ := json.Marshal(result)
	if strings.Contains(string(encoded), assertion) || strings.Contains(string(encoded), credential) {
		t.Fatal("token leaked")
	}
}

func TestWorkloadStopsAtDeniedIssuance(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid_client", "error_description": "DO NOT EXPOSE"})
	}))
	defer s.Close()
	result := workload{Client: s.Client(), TokenEndpoint: s.URL}.run("exchange")
	if result["token_status"] != 401 || result["exchange_status"] != nil {
		t.Fatal("denial not preserved")
	}
	b, _ := json.Marshal(result)
	if strings.Contains(string(b), "DO NOT EXPOSE") {
		t.Fatal("raw response leaked")
	}
}

func TestWorkloadRejectsUnknownModeWithoutNetwork(t *testing.T) {
	called := false
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
	defer s.Close()
	result := workload{Client: s.Client(), TokenEndpoint: s.URL}.run("anything")
	if called || result["error"] != "unsupported_mode" {
		t.Fatal("unsupported mode contacted a credential endpoint")
	}
}
