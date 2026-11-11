package main

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type trust struct {
	Issuer, Audience, Subject, ResourceAudience, STSIssuer string
	Keys                                                   map[string]*rsa.PublicKey
}

func newSTS(t trust, signingKey *rsa.PrivateKey) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /exchange", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
		if r.ParseForm() != nil || r.Form.Get("grant_type") != "urn:ietf:params:oauth:grant-type:token-exchange" || r.Form.Get("subject_token_type") != "urn:ietf:params:oauth:token-type:jwt" || r.Form.Get("audience") != t.ResourceAudience {
			deny(w, 400, "invalid_request")
			return
		}
		parsed, err := jwt.Parse(r.Form.Get("subject_token"), func(token *jwt.Token) (any, error) {
			kid, _ := token.Header["kid"].(string)
			key := t.Keys[kid]
			if key == nil {
				return nil, fmt.Errorf("unknown signing key")
			}
			return key, nil
		}, jwt.WithValidMethods([]string{"RS256"}), jwt.WithIssuer(t.Issuer), jwt.WithAudience(t.Audience), jwt.WithExpirationRequired(), jwt.WithIssuedAt())
		if err != nil {
			deny(w, 401, "invalid_subject_token")
			return
		}
		claims := parsed.Claims.(jwt.MapClaims)
		sub, _ := claims.GetSubject()
		aud, _ := claims.GetAudience()
		if sub != t.Subject || len(aud) != 1 {
			deny(w, 401, "invalid_subject_token")
			return
		}
		exp, _ := claims.GetExpirationTime()
		expiry := time.Now().Add(time.Minute)
		if exp.Time.Before(expiry) {
			expiry = exp.Time
		}
		token, err := signClaims(signingKey, "sts", jwt.MapClaims{"iss": t.STSIssuer, "aud": []string{t.ResourceAudience}, "sub": sub, "iat": time.Now().Unix(), "exp": expiry.Unix(), "permission": "read:fixture"})
		if err != nil {
			deny(w, 500, "issuance_failed")
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		respond(w, map[string]any{"access_token": token, "token_type": "Bearer", "issued_token_type": "urn:ietf:params:oauth:token-type:jwt"})
	})
	mux.HandleFunc("GET /resource", func(w http.ResponseWriter, r *http.Request) {
		parsed, err := jwt.Parse(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "), func(*jwt.Token) (any, error) { return &signingKey.PublicKey, nil }, jwt.WithValidMethods([]string{"RS256"}), jwt.WithIssuer(t.STSIssuer), jwt.WithAudience(t.ResourceAudience), jwt.WithExpirationRequired())
		if err != nil {
			deny(w, 401, "invalid_resource_token")
			return
		}
		claims := parsed.Claims.(jwt.MapClaims)
		if claims["permission"] != "read:fixture" || claims["sub"] != t.Subject {
			deny(w, 403, "permission_denied")
			return
		}
		respond(w, map[string]any{"authorized": true, "issuer": t.STSIssuer, "audience": t.ResourceAudience, "subject": t.Subject})
	})
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) { respond(w, map[string]any{"ready": true}) })
	return mux
}

func respond(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(value)
}
func deny(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": code})
}

func signClaims(key *rsa.PrivateKey, kid string, claims jwt.MapClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	return token.SignedString(key)
}
