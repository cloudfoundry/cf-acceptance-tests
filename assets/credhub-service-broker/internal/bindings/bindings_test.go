package bindings_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"code.cloudfoundry.org/credhub-cli/credhub"
	"code.cloudfoundry.org/credhub-cli/credhub/credentials"
	"code.cloudfoundry.org/credhub-cli/credhub/credentials/values"
	"code.cloudfoundry.org/credhub-cli/credhub/permissions"
	"github.com/cloudfoundry/cf-acceptance-tests/assets/credhub-service-broker/internal/bindings"
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
		body             string
		setJSONReturn    credentials.JSON
		setJSONReturnErr error
		addPermissionErr error
		expectedStatus   int
		expectedBody     string
		expectToBeSaved  bool
	}{
		{
			name:            "simple",
			body:            `{"app_guid": "test-app-guid"}`,
			setJSONReturn:   credentials.JSON{Base: credentials.Base{Name: "test-credhub"}},
			expectedStatus:  http.StatusCreated,
			expectedBody:    `{"credentials":{"credhub-ref":"test-credhub"}}`,
			expectToBeSaved: true,
		},
		{
			name:            "no-request-body",
			setJSONReturn:   credentials.JSON{Base: credentials.Base{Name: "test-credhub"}},
			expectedStatus:  http.StatusBadRequest,
			expectedBody:    "Failed to parse binding request: EOF",
			expectToBeSaved: false,
		},
		{
			name:            "no-app-guid-in-request-body",
			body:            `{}`,
			setJSONReturn:   credentials.JSON{Base: credentials.Base{Name: "test-credhub"}},
			expectedStatus:  http.StatusCreated,
			expectedBody:    `{"credentials":{"credhub-ref":"test-credhub"}}`,
			expectToBeSaved: true,
		},
		{
			name:             "fails-to-set-credhub-ref",
			body:             `{"app_guid": "test-app-guid"}`,
			setJSONReturnErr: fmt.Errorf("some error"),
			expectedStatus:   http.StatusInternalServerError,
			expectedBody:     "Failed to set credential: some error",
			expectToBeSaved:  false,
		},
		{
			name:             "fails-to-give-app-permissions-to-credhub-ref",
			body:             `{"app_guid": "test-app-guid"}`,
			setJSONReturn:    credentials.JSON{Base: credentials.Base{Name: "test-credhub"}},
			addPermissionErr: fmt.Errorf("some error"),
			expectedStatus:   http.StatusCreated,
			expectedBody:     `{"credentials":{"credhub-ref":"test-credhub"}}`,
			expectToBeSaved:  true,
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
			b := bindings.New(mcc)

			req := httptest.NewRequest("PUT", "/v2/service_instances/test-guid/service_bindings/test-binding-guid", strings.NewReader(tc.body))
			req.SetPathValue("binding_id", "test-binding-id")
			rr := httptest.NewRecorder()
			b.Add(rr, req)

			if status := rr.Code; status != tc.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tc.expectedStatus)
			}

			if rr.Body.String() != tc.expectedBody {
				t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), tc.expectedBody)
			}

			v, ok := b.Get("test-binding-id")
			if ok != tc.expectToBeSaved {
				t.Errorf("unexpected binding presence: got %v want %v", ok, tc.expectToBeSaved)
			}
			if ok && v != "test-credhub" {
				t.Errorf("unexpected binding value: got %v want %v", v, "test-credhub")
			}
		})
	}
}

func TestBindings_Remove(t *testing.T) {
	t.Parallel()

	mcc := &mockCredhubClient{}
	b := bindings.New(mcc)

	req, err := http.NewRequest("DELETE", "/v2/service_instances/test-guid/service_bindings/test-binding-guid", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	b.Remove(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	expected := `{}`
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}
}
