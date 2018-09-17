package credhub

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"time"
)

// Client provides an unauthenticated http.Client to the CredHub server
func (ch *CredHub) Client() *http.Client {
	if ch.defaultClient == nil {
		ch.defaultClient = ch.client()
	}

	return ch.defaultClient
}

func (ch *CredHub) client() *http.Client {
	if ch.baseURL.Scheme == "https" {
		return httpsClient(ch.insecureSkipVerify, ch.caCerts, ch.clientCertificate)
	}

	return httpClient()
}

func httpClient() *http.Client {
	return &http.Client{
		Timeout: time.Second * 45,
	}
}

func httpsClient(insecureSkipVerify bool, rootCAs *x509.CertPool, cert *tls.Certificate) *http.Client {
	client := httpClient()

	certs := []tls.Certificate{}
	if cert != nil {
		certs = []tls.Certificate{*cert}
	}

	client.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify:       insecureSkipVerify,
			PreferServerCipherSuites: true,
			Certificates:             certs,
			RootCAs:                  rootCAs,
		},
		Proxy: http.ProxyFromEnvironment,
	}

	return client
}
