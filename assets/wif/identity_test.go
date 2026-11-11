package main

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestIdentitySummaryPreservesOnlyPublicRuntimeIdentity(t *testing.T) {
	leaf := &x509.Certificate{DNSNames: []string{"instance", "worker.acme.svc.identity"}, Subject: pkix.Name{CommonName: "instance", OrganizationalUnit: []string{"app:app-guid", "space:space-guid", "organization:org-guid"}}, NotAfter: time.Now().Add(time.Hour), RawSubjectPublicKeyInfo: []byte("public-key")}
	s := identitySummary(`{"client_id":"cf:service-account:acme:worker","token_endpoint":"https://uaa.example/token","secret":"never-return"}`, leaf)
	if s["client_id"] != "cf:service-account:acme:worker" || s["app_guid"] != "app-guid" || s["space_guid"] != "space-guid" || s["org_guid"] != "org-guid" || s["cf_instance_guid"] != "instance" {
		t.Fatal("missing attribution")
	}
	if s["certificate_present"] != true || s["account_discovery_present"] != true || s["public_key_sha256"] == nil {
		t.Fatal("missing credential observations")
	}
	b, _ := json.Marshal(s)
	if strings.Contains(string(b), "never-return") || strings.Contains(string(b), "token_endpoint") {
		t.Fatal("raw discovery leaked")
	}
}

func TestIdentitySummaryObservesStagingAbsence(t *testing.T) {
	s := identitySummary("", nil)
	if s["account_discovery_present"] != false || s["certificate_present"] != false {
		t.Fatal("missing explicit absence")
	}
}
