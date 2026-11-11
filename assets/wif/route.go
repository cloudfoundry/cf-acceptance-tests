package main

import "net/http"

type routeProbe struct {
	Targets map[string]string
	Client func(bool) *http.Client
	SpoofedSAN string
}

func (p routeProbe) handler() http.Handler { return http.NotFoundHandler() }
