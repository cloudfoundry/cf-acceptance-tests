package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGreetingIdentifiesAppAndInstanceAtEachContextPath(t *testing.T) {
	for _, app := range []struct{ name, index string }{{"first-app", "0"}, {"second-app", "2"}} {
		t.Run(app.name, func(t *testing.T) {
			t.Setenv("VCAP_APPLICATION", fmt.Sprintf(`{"name":%q}`, app.name))
			t.Setenv("CF_INSTANCE_INDEX", app.index)
			for _, path := range []string{"/", "/app1", "/app2", "/app3/long/sub/path"} {
				t.Run(path, func(t *testing.T) {
					res := httptest.NewRecorder()
					greet(res, httptest.NewRequest(http.MethodGet, path, nil))
					if res.Code != http.StatusOK {
						t.Fatalf("status = %d, want 200", res.Code)
					}
					want := fmt.Sprintf("Hello, %s at index: %s!", app.name, app.index)
					if res.Body.String() != want {
						t.Errorf("body = %q, want %q", res.Body.String(), want)
					}
					if got := res.Header().Get("Set-Cookie"); got != "JSESSIONID=12345" {
						t.Errorf("cookie = %q, want JSESSIONID=12345 without a Path attribute", got)
					}
					if got := res.Header().Get("Content-Type"); got != "text/plain" {
						t.Errorf("content type = %q, want text/plain", got)
					}
				})
			}
		})
	}
}

func TestInvalidApplicationMetadataReturnsServerError(t *testing.T) {
	t.Setenv("VCAP_APPLICATION", "invalid JSON")
	res := httptest.NewRecorder()
	greet(res, httptest.NewRequest(http.MethodGet, "/", nil))
	if res.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", res.Code)
	}
}
