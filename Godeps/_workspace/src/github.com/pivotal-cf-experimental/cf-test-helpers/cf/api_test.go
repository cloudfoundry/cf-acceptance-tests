package cf_test

import (
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
	"github.com/pivotal-cf-experimental/cf-test-helpers/runner"
	"github.com/vito/cmdtest"
)

var _ = Describe("ApiRequest", func() {
	It("sends the request to current CF target", func() {
		runner.SessionStarter = func(cmd *exec.Cmd) (*cmdtest.Session, error) {
			Expect(cmd.Path).To(Equal(exec.Command("gcf").Path))
			Expect(cmd.Args).To(Equal([]string{
				"gcf", "curl", "/v2/info", "-X", "GET", "-d", "somedata",
			}))

			return cmdtest.Start(exec.Command("bash", "-c", `echo \{ \"metadata\": \{ \"guid\": \"abc\" \} \}`))
		}

		var response GenericResource
		ApiRequest("GET", "/v2/info", &response, "some", "data")

		Expect(response.Metadata.Guid).To(Equal("abc"))
	})
})
