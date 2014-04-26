package cf

import (
	"encoding/json"
	"strings"

	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
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

	Eventually(request).Should(Exit(0))

	if response != nil {
		err := json.Unmarshal(request.Out.Contents(), response)
		Expect(err).ToNot(HaveOccurred())
	}
}
