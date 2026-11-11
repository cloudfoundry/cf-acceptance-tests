package main

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type workload struct {
	Client                                                *http.Client
	TokenEndpoint, ClientID, Resource, STSURL, CAPITarget string
}

func (w workload) run(mode string) map[string]any {
	result := map[string]any{}
	if mode != "exchange" && mode != "default-audience" && mode != "unlisted" && mode != "bearer" {
		result["error"] = "unsupported_mode"
		return result
	}
	form := url.Values{"grant_type": {"client_credentials"}, "client_id": {w.ClientID}, "token_format": {"jwt"}}
	if mode != "default-audience" && mode != "bearer" {
		resource := w.Resource
		if mode == "unlisted" {
			resource = "urn:cats:unlisted"
		}
		form.Set("resource", resource)
	}
	resp, err := w.Client.PostForm(w.TokenEndpoint, form)
	if err != nil {
		result["error"] = "token_transport"
		return result
	}
	result["token_status"] = resp.StatusCode
	var token struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	err = json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&token)
	resp.Body.Close()
	if err != nil || token.AccessToken == "" {
		result["token_error"] = token.Error
		return result
	}
	// The UAA token stays inside this process; never return it or its raw body.
	if mode == "bearer" {
		parts := strings.Split(token.AccessToken, ".")
		if len(parts) != 3 {
			result["error"] = "invalid_jwt"
			return result
		}
		payload, err := base64.RawURLEncoding.DecodeString(parts[1])
		if err != nil {
			result["error"] = "invalid_jwt"
			return result
		}
		var claims map[string]any
		if json.Unmarshal(payload, &claims) != nil {
			result["error"] = "invalid_jwt"
			return result
		}
		for _, key := range []string{"sub", "iss", "aud", "iat", "exp", "app_guid", "space_guid", "org_guid", "cf_instance_guid"} {
			result[key] = claims[key]
		}
		_, result["cnf_present"] = claims["cnf"]
		req, _ := http.NewRequest("GET", w.CAPITarget+"/v3/apps?per_page=500", nil)
		req.Header.Set("Authorization", "Bearer "+token.AccessToken)
		api, err := w.Client.Do(req)
		if err != nil {
			result["error"] = "capi_transport"
			return result
		}
		defer api.Body.Close()
		result["capi_status"] = api.StatusCode
		var apps struct {
			Resources []struct {
				GUID string `json:"guid"`
			} `json:"resources"`
		}
		if json.NewDecoder(io.LimitReader(api.Body, 1<<20)).Decode(&apps) == nil {
			guids := []string{}
			for _, app := range apps.Resources {
				guids = append(guids, app.GUID)
			}
			result["visible_apps"] = guids
		}
		return result
	}
	resourceStatus := func(bearer string) int {
		req, _ := http.NewRequest("GET", w.STSURL+"/resource", nil)
		req.Header.Set("Authorization", "Bearer "+bearer)
		r, e := w.Client.Do(req)
		if e != nil {
			return 0
		}
		defer r.Body.Close()
		var resource struct {
			Authorized bool   `json:"authorized"`
			Issuer     string `json:"issuer"`
			Audience   string `json:"audience"`
			Subject    string `json:"subject"`
		}
		if json.NewDecoder(io.LimitReader(r.Body, 64<<10)).Decode(&resource) == nil && r.StatusCode == 200 {
			result["authorized"] = resource.Authorized
			result["resource_issuer"] = resource.Issuer
			result["resource_audience"] = resource.Audience
			result["resource_subject"] = resource.Subject
		}
		return r.StatusCode
	}
	result["direct_resource_status"] = resourceStatus(token.AccessToken)
	exchange, err := w.Client.PostForm(w.STSURL+"/exchange", url.Values{"grant_type": {"urn:ietf:params:oauth:grant-type:token-exchange"}, "subject_token_type": {"urn:ietf:params:oauth:token-type:jwt"}, "subject_token": {token.AccessToken}, "audience": {"urn:cats:resource"}})
	if err != nil {
		result["error"] = "exchange_transport"
		return result
	}
	result["exchange_status"] = exchange.StatusCode
	var exchanged struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	err = json.NewDecoder(io.LimitReader(exchange.Body, 64<<10)).Decode(&exchanged)
	exchange.Body.Close()
	if err != nil || exchanged.AccessToken == "" {
		result["exchange_error"] = exchanged.Error
		return result
	}
	result["resource_status"] = resourceStatus(exchanged.AccessToken)
	req, _ := http.NewRequest("GET", w.CAPITarget+"/v3/apps", nil)
	req.Header.Set("Authorization", "Bearer "+exchanged.AccessToken)
	api, err := w.Client.Do(req)
	if err == nil {
		result["exchanged_capi_status"] = api.StatusCode
		api.Body.Close()
	}
	return result
}
