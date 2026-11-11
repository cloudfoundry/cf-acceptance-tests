package config_test

import (
	cfg "github.com/cloudfoundry/cf-acceptance-tests/helpers/config"
	"testing"
)

func TestWIFPrerequisites(t *testing.T) {
	if cfg.ValidateWIF(cfg.WIFConfig{}) != nil {
		t.Fatal("disabled WIF requires configuration")
	}
	valid := cfg.WIFConfig{Enabled: true, Issuer: "https://uaa.example/oauth/token", JWKSURI: "https://uaa.example/token_keys", Audience: "urn:cats:federation"}
	if err := cfg.ValidateWIF(valid); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"issuer", "jwks", "audience"} {
		t.Run(field, func(t *testing.T) {
			c := valid
			switch field {
			case "issuer":
				c.Issuer = ""
			case "jwks":
				c.JWKSURI = "http://untrusted.example/keys"
			case "audience":
				c.Audience = "not a URI"
			}
			if cfg.ValidateWIF(c) == nil {
				t.Fatal("invalid WIF prerequisite accepted")
			}
		})
	}
}
