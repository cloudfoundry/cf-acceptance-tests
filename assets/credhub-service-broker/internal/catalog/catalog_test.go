package catalog_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudfoundry/cf-acceptance-tests/assets/credhub-service-broker/internal/catalog"
)

func TestCatalog_Handler(t *testing.T) {
	t.Parallel()

	c := catalog.New("test-name", "test-id", "test-plan-id")

	rr := httptest.NewRecorder()
	c.ServeHTTP(rr, nil)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	expected := `{"services":[{"name":"test-name","id":"test-id","description":"credhub read service for tests","bindable":true,"plans":[{"name":"credhub-read-plan","id":"test-plan-id","description":"credhub read plan for tests"}]}]}`
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}
}
