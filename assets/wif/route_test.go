package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouteProbeUsesOnlyConfiguredTargetAndNeverBearer(t *testing.T) {
	calls := 0
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "" {
			t.Error("unexpected bearer")
		}
		if r.Header.Get("X-Forwarded-Client-Cert") == "" {
			t.Error("missing spoofed header")
		}
		w.WriteHeader(403)
	}))
	defer backend.Close()
	withCert := false
	p := routeProbe{Targets: map[string]string{"account": backend.URL}, Client: func(sent bool) *http.Client { withCert = sent; return backend.Client() }, SpoofedSAN: "worker.acme.svc.identity"}
	s := httptest.NewServer(p.handler())
	defer s.Close()
	r, err := http.Get(s.URL + "/route?target=account&mode=spoof")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	var result map[string]any
	json.NewDecoder(r.Body).Decode(&result)
	if result["status"] != float64(403) || !withCert || calls != 1 {
		t.Fatal("route request not forwarded correctly")
	}
	r2, err := http.Get(s.URL + "/route?target=https://evil.example&mode=normal")
	if err != nil {
		t.Fatal(err)
	}
	r2.Body.Close()
	if r2.StatusCode != 400 || calls != 1 {
		t.Fatal("unconfigured target allowed")
	}
}

func TestRouteProbeNoCertificateAndIdentityIndependentHealth(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	defer backend.Close()
	withCert := true
	s := httptest.NewServer(routeProbe{Targets: map[string]string{"account": backend.URL}, Client: func(sent bool) *http.Client { withCert = sent; return backend.Client() }}.handler())
	defer s.Close()
	r, err := http.Get(s.URL + "/route?target=account&mode=no-cert")
	if err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	if r.StatusCode != 200 || withCert {
		t.Fatal("no-cert mode still sent certificate")
	}
	r, err = http.Get(s.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	if r.StatusCode != 200 {
		t.Fatal("unbound backend unavailable")
	}
}
