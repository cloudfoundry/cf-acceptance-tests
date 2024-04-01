package bindings

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"code.cloudfoundry.org/credhub-cli/credhub"
	"code.cloudfoundry.org/credhub-cli/credhub/credentials"
	"code.cloudfoundry.org/credhub-cli/credhub/credentials/values"
	"code.cloudfoundry.org/credhub-cli/credhub/permissions"
)

type CredhubClient interface {
	SetJSON(name string, value values.JSON, options ...credhub.SetOption) (credentials.JSON, error)
	AddPermission(path string, actor string, ops []string) (*permissions.Permission, error)
	Delete(name string) error
}

type Bindings struct {
	m  map[string]string
	cc CredhubClient
}

func New(cc CredhubClient) *Bindings {
	return &Bindings{
		m:  make(map[string]string),
		cc: cc,
	}
}

type BindingRequest struct {
	AppGuid string `json:"app_guid"`
}

type Credentials struct {
	CredhubRef string `json:"credhub-ref"`
}

type BindingResponse struct {
	Credentials Credentials `json:"credentials"`
}

func (b *Bindings) Add(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("binding_id")
	if id == "" {
		log.Panicf("Path does not include binding id")
	}

	var br BindingRequest
	err := json.NewDecoder(r.Body).Decode(&br)
	r.Body.Close()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Failed to parse binding request: %s", err.Error())))
		return
	}

	name := strconv.FormatInt(time.Now().UnixNano(), 10)
	cred, err := b.cc.SetJSON(name, values.JSON{
		"user-name": "pinkyPie",
		"password":  "rainbowDash",
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("Failed to set credential: %s", err.Error())))
		return
	}

	b.m[id] = cred.Name

	if br.AppGuid != "" {
		_, err = b.cc.AddPermission(cred.Name, "mtls-app:"+br.AppGuid, []string{"read"})
		if err != nil {
			log.Println("Failed to add permission: ", err)
		}
	}

	resp := BindingResponse{
		Credentials: Credentials{
			CredhubRef: "test-credhub",
		},
	}
	respJSON, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("Failed to marshal binding response: %s", err.Error())))
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write(respJSON)
}

func (b *Bindings) Remove(w http.ResponseWriter, r *http.Request) {
	// id := r.PathValue("id")

	w.Write([]byte("{}"))
}

func (b *Bindings) Get(id string) (string, bool) {
	resp, ok := b.m[id]
	return resp, ok
}

func (b Bindings) Register(mux *http.ServeMux) {
	mux.HandleFunc("/v2/service_instances/{id}/service_bindings/{binding_id}", b.Add)
	mux.HandleFunc("/v2/service_instances/{id}/service_bindings/{binding_id}", b.Remove)
}
