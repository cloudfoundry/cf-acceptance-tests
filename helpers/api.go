package helpers

import (
	"encoding/json"
	"fmt"
	"strings"

	. "github.com/onsi/gomega"
	. "github.com/vito/cmdtest/matchers"
)

type GenericResource struct {
	Metadata struct {
		Guid string `json:"guid"`
	} `json:"metadata"`
}

type QueryResponse struct {
	Resources []GenericResource `struct:"resources"`
}

func ApiRequest(method, endpoint string, response interface{}, data ...string) {
	config := LoadCfConfig()

	request := Curl(
		"-k",
		config.Target+endpoint,
		"-X", method,
		"-d", strings.Join(data, ""),
		"-H", fmt.Sprintf("Authorization: %s", config.AccessToken),
	)

	Expect(request).To(ExitWith(0))

	if response != nil {
		err := json.Unmarshal(request.FullOutput(), response)
		Expect(err).ToNot(HaveOccurred())
	}
}
