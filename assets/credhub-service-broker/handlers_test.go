package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"code.cloudfoundry.org/credhub-cli/credhub"
	"code.cloudfoundry.org/credhub-cli/credhub/credentials"
	"code.cloudfoundry.org/credhub-cli/credhub/credentials/values"
	"code.cloudfoundry.org/credhub-cli/credhub/permissions"
)

type mockCredhubClient struct {
	SetJSONReturn          credentials.JSON
	SetJSONReturnErr       error
	AddPermissionReturn    *permissions.Permission
	AddPermissionReturnErr error
}

func (m *mockCredhubClient) SetJSON(name string, value values.JSON, options ...credhub.SetOption) (credentials.JSON, error) {
	return m.SetJSONReturn, m.SetJSONReturnErr
}

func (m *mockCredhubClient) AddPermission(path string, actor string, ops []string) (*permissions.Permission, error) {
	return m.AddPermissionReturn, m.AddPermissionReturnErr
}

func (m *mockCredhubClient) Delete(name string) error {
	return nil
}

func TestBindings_Add(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		bindingID        string
		body             string
		setJSONReturn    credentials.JSON
		setJSONReturnErr error
		expectedStatus   int
		expectedBody     string
	}{
		{
			name:           "simple",
			bindingID:      "test-binding-id",
			body:           `{"app_guid": "test-app-guid"}`,
			setJSONReturn:  credentials.JSON{Base: credentials.Base{Name: "test-credhub"}},
			expectedStatus: http.StatusCreated,
			expectedBody:   `{"credentials":{"credhub-ref":"test-credhub"}}`,
		},
		{
			name:           "no-service-binding-id",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mcc := &mockCredhubClient{
				SetJSONReturn:    tc.setJSONReturn,
				SetJSONReturnErr: tc.setJSONReturnErr,
			}

			req := httptest.NewRequest("PUT", "/v2/service_instances/test-guid/service_bindings/test-binding-guid", strings.NewReader(tc.body))
			req.SetPathValue("binding_id", tc.bindingID)

			rr := httptest.NewRecorder()
			hf := bindHandler(mcc, map[string]string{})
			hf.ServeHTTP(rr, req)

			if status := rr.Code; status != tc.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tc.expectedStatus)
			}

			if rr.Body.String() != tc.expectedBody {
				t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), tc.expectedBody)
			}
		})
	}
}
