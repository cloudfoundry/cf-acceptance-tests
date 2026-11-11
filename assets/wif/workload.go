package main

import "net/http"

type workload struct {
	Client *http.Client
	TokenEndpoint, ClientID, Resource, STSURL, CAPITarget string
}

func (w workload) run(mode string) map[string]any { return map[string]any{} }
