package cf_test

import (
	"fmt"
	"os"
	"os/exec"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
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

	Context("when CF_TRACE_BASENAME is set", func() {
		BeforeEach(func() {
			os.Setenv("CF_TRACE_BASENAME", "/big-mouth-billy-base")
		})

		It("sets CF_TRACE", func() {
			runner.SessionStarter = func(cmd *exec.Cmd) (*cmdtest.Session, error) {
				Expect(os.Getenv("CF_TRACE")).To(Equal(
					fmt.Sprintf("/big-mouth-billy-base%d.txt", config.GinkgoConfig.ParallelNode),
				))

				return cmdtest.Start(exec.Command("bash", "-c", `exit 42`))
			}

			session := Cf("apps")
			Expect(session).To(ExitWith(42))
		})
	})
})
