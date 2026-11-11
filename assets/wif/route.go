package main

import (
	"io"
	"net/http"
)

type routeProbe struct {
	Targets    map[string]string
	Client     func(bool) *http.Client
	SpoofedSAN string
}

func (p routeProbe) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) { respond(w, map[string]bool{"ready": true}) })
	mux.HandleFunc("GET /route", func(w http.ResponseWriter, r *http.Request) {
		target := p.Targets[r.URL.Query().Get("target")]
		mode := r.URL.Query().Get("mode")
		if target == "" || (mode != "normal" && mode != "spoof" && mode != "no-cert") {
			deny(w, 400, "unsupported_route_request")
			return
		}
		req, err := http.NewRequest("GET", target, nil)
		if err != nil {
			deny(w, 500, "invalid_target_configuration")
			return
		}
		if mode == "spoof" {
			req.Header.Set("X-Forwarded-Client-Cert", `Hash=fake;Subject="CN=fake,OU=app:fake";DNS=`+p.SpoofedSAN)
		}
		result := map[string]any{"certificate_sent": mode != "no-cert"}
		resp, err := p.Client(mode != "no-cert").Do(req)
		if err != nil {
			result["tls_error"] = true
			respond(w, result)
			return
		}
		defer resp.Body.Close()
		io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
		result["status"] = resp.StatusCode
		respond(w, result)
	})
	return mux
}
