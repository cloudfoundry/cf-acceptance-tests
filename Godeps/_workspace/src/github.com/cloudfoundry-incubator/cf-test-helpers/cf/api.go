package cf

import (
	"encoding/json"
	"strings"
	"time"

	. "github.com/onsi/gomega"

	"github.com/cloudfoundry-incubator/cf-test-helpers/runner"
)

//var CfApiTimeout = 30 * time.Second

type GenericResource struct {
	Metadata struct {
		Guid string `json:"guid"`
	} `json:"metadata"`
}

type QueryResponse struct {
	Resources []GenericResource `struct:"resources"`
}

var ApiRequest = func(method, endpoint string, response interface{}, timeout time.Duration, data ...string) {
	args := []string{
		"curl",
		endpoint,
		"-X", method,
	}

	dataArg := strings.Join(data, "")
	if len(dataArg) > 0 {
		args = append(args, "-d", dataArg)
	}

	request := Cf(args...)
	runner.NewCmdRunner(request, timeout).Run()

	if response != nil {
		err := json.Unmarshal(request.Out.Contents(), response)
		Expect(err).ToNot(HaveOccurred())
	}

}
