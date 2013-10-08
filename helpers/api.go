package helpers

import (
	"encoding/json"
	"fmt"
	"strings"

	. "github.com/onsi/gomega"
	. "github.com/vito/cmdtest/matchers"
)

type AppResource struct {
	Metadata struct {
		Guid string `json:"guid"`
	} `json:"metadata"`
}

type AppQueryResponse struct {
	Resources []AppResource `struct:"resources"`
}

func ApiRequest(method, endpoint string, response interface{}, data ...string) {
	config := LoadCfConfig()

	request := Curl(
		config.Target + endpoint,
		"-X", method,
		"-d", strings.Join(data, ""),
		"-H", fmt.Sprintf("Authorization: %s", config.AccessToken),
	)

	Expect(request).To(ExitWith(0))

	if response != nil {
		err := json.Unmarshal(request.FullOutput(), response)
		Expect(err).ToNot(HaveOccured())
	}
}
