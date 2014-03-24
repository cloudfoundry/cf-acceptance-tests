package cf

import (
	"encoding/json"
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
	request := Cf(
		"curl",
		endpoint,
		"-X", method,
		"-d", strings.Join(data, ""),
	)

	Expect(request).To(ExitWith(0))

	if response != nil {
		err := json.Unmarshal(request.FullOutput(), response)
		Expect(err).ToNot(HaveOccurred())
	}
}
