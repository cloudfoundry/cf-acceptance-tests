package cf_test

import (
	"os/exec"
	"time"

	. "github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/runner"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ApiRequest", func() {
	It("sends the request to current CF target", func() {
		runner.CommandInterceptor = func(cmd *exec.Cmd) *exec.Cmd {
			Expect(cmd.Path).To(Equal(exec.Command("cf").Path))
			Expect(cmd.Args).To(Equal([]string{
				"cf", "curl", "/v2/info", "-X", "GET", "-d", "somedata",
			}))

			return exec.Command("bash", "-c", `echo \{ \"metadata\": \{ \"guid\": \"abc\" \} \}`)
		}

		var response GenericResource
		ApiRequest("GET", "/v2/info", &response, 1*time.Second, "some", "data")

		Expect(response.Metadata.Guid).To(Equal("abc"))
	})
})
