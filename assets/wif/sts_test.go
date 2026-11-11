package main

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestFederationContract(t *testing.T) {
	uaa, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil { t.Fatal(err) }
	stsKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil { t.Fatal(err) }
	policy := trust{Issuer:"https://uaa.example", Audience:"https://federation.example", Subject:"cf:service-account:acme:worker", ResourceAudience:"urn:cats:resource", STSIssuer:"urn:cats:sts", Keys:map[string]*rsa.PublicKey{"uaa-key":&uaa.PublicKey}}
	server := httptest.NewServer(newSTS(policy, stsKey))
	defer server.Close()
	claims := func() jwt.MapClaims { return jwt.MapClaims{"iss":policy.Issuer,"aud":[]string{policy.Audience},"sub":policy.Subject,"exp":time.Now().Add(45*time.Second).Unix(),"iat":time.Now().Unix()} }
	exchange := func(assertion string) (*http.Response, []byte) {
		t.Helper()
		resp, err := http.PostForm(server.URL+"/exchange", url.Values{"grant_type":{"urn:ietf:params:oauth:grant-type:token-exchange"},"subject_token_type":{"urn:ietf:params:oauth:token-type:jwt"},"subject_token":{assertion},"audience":{policy.ResourceAudience}})
		if err != nil { t.Fatal(err) }
		body, err := io.ReadAll(resp.Body); resp.Body.Close(); if err != nil { t.Fatal(err) }; return resp,body
	}
	t.Run("exchanges verified UAA identity for a distinct, expiry-capped resource credential",func(t *testing.T){
		input := claims()
		assertion, err := signClaims(uaa,"uaa-key",input); if err != nil {t.Fatal(err)}
		resp, body := exchange(assertion)
		if resp.StatusCode != 200 {t.Fatalf("exchange status %d",resp.StatusCode)}
		var result struct{ AccessToken string `json:"access_token"` }; if err:=json.Unmarshal(body,&result);err!=nil{t.Fatal(err)}
		parsed,err:=jwt.Parse(result.AccessToken,func(*jwt.Token)(any,error){return &stsKey.PublicKey,nil},jwt.WithValidMethods([]string{"RS256"}),jwt.WithIssuer(policy.STSIssuer),jwt.WithAudience(policy.ResourceAudience),jwt.WithExpirationRequired())
		if err!=nil{t.Fatal(err)}
		out:=parsed.Claims.(jwt.MapClaims)
		if out["sub"]!=policy.Subject || out["permission"]!="read:fixture" || out["exp"].(float64)>float64(input["exp"].(int64)){t.Fatal("wrong exchanged claims")}
		for _, tc:=range []struct{token string;status int}{{result.AccessToken,200},{assertion,401}}{
			req,_:=http.NewRequest("GET",server.URL+"/resource",nil);req.Header.Set("Authorization","Bearer "+tc.token)
			r,err:=http.DefaultClient.Do(req);if err!=nil{t.Fatal(err)};b,_:=io.ReadAll(r.Body);r.Body.Close()
			if r.StatusCode!=tc.status{t.Fatalf("resource status %d, want %d",r.StatusCode,tc.status)}
			if strings.Contains(string(b),tc.token){t.Fatal("resource leaked token")}
		}
	})
	for _,field:=range []string{"iss","aud","sub","exp","nbf","signature","algorithm","kid"}{
		t.Run("rejects_"+field,func(t *testing.T){
			c:=claims();key:=uaa;kid:="uaa-key"
			switch field{case "iss":c["iss"]="https://other.example";case "aud":c["aud"]="cloud_controller";case "sub":c["sub"]="cf:service-account:acme:other";case "exp":c["exp"]=time.Now().Add(-time.Minute).Unix();case "nbf":c["nbf"]=time.Now().Add(time.Hour).Unix();case "signature":key=stsKey;case "kid":kid="unknown"}
			assertion,err:=signClaims(key,kid,c);if field=="algorithm"{assertion,err=jwt.NewWithClaims(jwt.SigningMethodHS256,c).SignedString([]byte("wrong-key"))};if err!=nil{t.Fatal(err)}
			r,b:=exchange(assertion);if r.StatusCode!=401{t.Fatalf("status %d, want 401",r.StatusCode)}
			if strings.Contains(string(b),assertion)||strings.Contains(string(b),"access_token"){t.Fatal("denial leaked credentials")}
		})
	}
}
