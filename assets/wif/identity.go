package main

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"
)

func identitySummary(discovery string, leaf *x509.Certificate) map[string]any {
	r := map[string]any{"account_discovery_present": discovery != "", "certificate_present": leaf != nil}
	var d struct {
		ClientID string `json:"client_id"`
	}
	if json.Unmarshal([]byte(discovery), &d) == nil {
		r["client_id"] = d.ClientID
	}
	if leaf == nil {
		return r
	}
	r["dns_sans"] = leaf.DNSNames
	r["cf_instance_guid"] = leaf.Subject.CommonName
	r["certificate_not_after"] = leaf.NotAfter.UTC().Format(time.RFC3339)
	h := sha256.Sum256(leaf.RawSubjectPublicKeyInfo)
	r["public_key_sha256"] = hex.EncodeToString(h[:])
	for _, ou := range leaf.Subject.OrganizationalUnit {
		for prefix, claim := range map[string]string{"app:": "app_guid", "space:": "space_guid", "organization:": "org_guid"} {
			if strings.HasPrefix(ou, prefix) {
				r[claim] = strings.TrimPrefix(ou, prefix)
			}
		}
	}
	return r
}
