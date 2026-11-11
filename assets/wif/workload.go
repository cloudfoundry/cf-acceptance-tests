package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
)

type workload struct {
	Client                                                *http.Client
	TokenEndpoint, ClientID, Resource, STSURL, CAPITarget string
}

func (w workload) run(mode string) map[string]any {
	result := map[string]any{}
	form := url.Values{"grant_type": {"client_credentials"}, "client_id": {w.ClientID}, "token_format": {"jwt"}}
	if mode != "default-audience" {
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
	resourceStatus := func(bearer string) int {
		req, _ := http.NewRequest("GET", w.STSURL+"/resource", nil)
		req.Header.Set("Authorization", "Bearer "+bearer)
		r, e := w.Client.Do(req)
		if e != nil {
			return 0
		}
		defer r.Body.Close()
		io.Copy(io.Discard, io.LimitReader(r.Body, 64<<10))
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
