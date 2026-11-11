package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"time"
)

func decodeURL(s string) ([]byte, error) { return base64.RawURLEncoding.DecodeString(s) }

func httpClient(certificates []tls.Certificate) *http.Client {
	roots, _ := x509.SystemCertPool()
	if roots == nil {
		roots = x509.NewCertPool()
	}
	if extra := os.Getenv("TRUSTED_CA_CERT"); extra != "" && !roots.AppendCertsFromPEM([]byte(extra)) {
		panic("invalid trust roots")
	}
	return &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("redirect refused") }, Transport: &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: roots, Certificates: certificates}}}
}

func loadKeys(client *http.Client, endpoint string) (map[string]*rsa.PublicKey, error) {
	r, err := client.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("JWKS fetch failed: %w", err)
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		return nil, errors.New("JWKS status failed")
	}
	var set struct {
		Keys []struct {
			Kid   string `json:"kid"`
			Kty   string `json:"kty"`
			Value string `json:"value"`
			N     string `json:"n"`
			E     string `json:"e"`
		} `json:"keys"`
	}
	if json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&set) != nil {
		return nil, errors.New("JWKS decode failed")
	}
	keys := map[string]*rsa.PublicKey{}
	for _, k := range set.Keys {
		if k.Kty != "RSA" {
			continue
		}
		if block, _ := pem.Decode([]byte(k.Value)); block != nil {
			pub, e := x509.ParsePKIXPublicKey(block.Bytes)
			if e == nil {
				if key, ok := pub.(*rsa.PublicKey); ok {
					keys[k.Kid] = key
				}
			}
		} else {
			n, e1 := decodeURL(k.N)
			e, e2 := decodeURL(k.E)
			if e1 == nil && e2 == nil && len(e) <= 4 {
				keys[k.Kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(new(big.Int).SetBytes(e).Int64())}
			}
		}
	}
	if len(keys) == 0 {
		return nil, errors.New("no RSA verification keys")
	}
	return keys, nil
}

func main() {
	var handler http.Handler
	if os.Getenv("WIF_MODE") == "route" {
		handler = routeProbe{Targets: map[string]string{"account": os.Getenv("ACCOUNT_ROUTE_URL"), "space": os.Getenv("SPACE_ROUTE_URL")}, SpoofedSAN: os.Getenv("SPOOFED_ACCOUNT_SAN"), Client: func(sent bool) *http.Client {
			var certs []tls.Certificate
			if sent {
				cert, err := tls.LoadX509KeyPair(os.Getenv("CF_INSTANCE_CERT"), os.Getenv("CF_INSTANCE_KEY"))
				if err != nil {
					panic("credentials unavailable")
				}
				certs = []tls.Certificate{cert}
			}
			return httpClient(certs)
		}}.handler()
	} else if os.Getenv("WIF_MODE") == "sts" {
		keys, err := loadKeys(httpClient(nil), os.Getenv("TRUSTED_JWKS_URI"))
		if err != nil {
			panic(err)
		}
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			panic("key generation failed")
		}
		handler = newSTS(trust{Issuer: os.Getenv("TRUSTED_ISSUER"), Audience: os.Getenv("TRUSTED_FEDERATION_AUDIENCE"), Subject: os.Getenv("TRUSTED_SERVICE_ACCOUNT_SUBJECT"), ResourceAudience: "urn:cats:resource", STSIssuer: "urn:cats:sts", Keys: keys}, key)
	} else {
		mux := http.NewServeMux()
		mux.HandleFunc("GET /run", func(w http.ResponseWriter, r *http.Request) {
			// Reload credentials and platform discovery on every request.
			var identity struct {
				ClientID      string `json:"client_id"`
				TokenEndpoint string `json:"token_endpoint"`
			}
			if json.Unmarshal([]byte(os.Getenv("VCAP_SERVICE_ACCOUNT")), &identity) != nil || identity.ClientID == "" {
				deny(w, 500, "account_discovery_missing")
				return
			}
			cert, err := tls.LoadX509KeyPair(os.Getenv("CF_INSTANCE_CERT"), os.Getenv("CF_INSTANCE_KEY"))
			if err != nil {
				deny(w, 500, "credentials_unavailable")
				return
			}
			mode := r.URL.Query().Get("mode")
			if mode != "exchange" && mode != "default-audience" && mode != "unlisted" {
				deny(w, 400, "unsupported_mode")
				return
			}
			client := httpClient([]tls.Certificate{cert})
			// Only UAA gets the workload certificate. STS/CAPI calls use plain bearer transports.
			transport := client.Transport
			client.Transport = endpointTransport{token: identity.TokenEndpoint, mtls: transport, plain: httpClient(nil).Transport}
			result := workload{Client: client, TokenEndpoint: identity.TokenEndpoint, ClientID: identity.ClientID, Resource: os.Getenv("TRUSTED_FEDERATION_AUDIENCE"), STSURL: os.Getenv("STS_URL"), CAPITarget: os.Getenv("CAPI_URL")}.run(mode)
			result["client_id"] = identity.ClientID
			respond(w, result)
		})
		handler = mux
	}
	fmt.Println("WIF fixture ready")
	server := &http.Server{Addr: ":" + os.Getenv("PORT"), Handler: handler, ReadHeaderTimeout: 10 * time.Second}
	if err := server.ListenAndServe(); err != nil {
		panic("HTTP server stopped")
	}
}

type endpointTransport struct {
	token       string
	mtls, plain http.RoundTripper
}

func (t endpointTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.String() == t.token {
		return t.mtls.RoundTrip(r)
	}
	return t.plain.RoundTrip(r)
}
