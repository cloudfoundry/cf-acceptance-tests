package main

import (
	"crypto/rsa"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
)

type trust struct {
	Issuer, Audience, Subject, ResourceAudience, STSIssuer string
	Keys map[string]*rsa.PublicKey
}

func newSTS(t trust, signingKey *rsa.PrivateKey) http.Handler {
	return http.NotFoundHandler()
}

func signClaims(key *rsa.PrivateKey, kid string, claims jwt.MapClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	return token.SignedString(key)
}
