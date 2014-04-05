package cf_test

import (
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
	"github.com/pivotal-cf-experimental/cf-test-helpers/runner"
	"github.com/vito/cmdtest"
	. "github.com/vito/cmdtest/matchers"
)

var _ = Describe("Cf", func() {
	It("sends the request to current CF target", func() {
		runner.SessionStarter = func(cmd *exec.Cmd) (*cmdtest.Session, error) {
			Expect(cmd.Path).To(Equal(exec.Command("gcf").Path))
			Expect(cmd.Args).To(Equal([]string{"gcf", "apps"}))

			return cmdtest.Start(exec.Command("bash", "-c", `exit 42`))
		}

		Expect(Cf("apps")).To(ExitWith(42))
	})
})
