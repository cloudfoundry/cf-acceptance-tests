package runner_test

import (
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf-experimental/cf-test-helpers/runner"
	"github.com/vito/cmdtest"
	. "github.com/vito/cmdtest/matchers"
)

var originalStarter = runner.SessionStarter

var _ = AfterEach(func() {
	runner.SessionStarter = originalStarter
})

var _ = Describe("Run", func() {
	It("runs the given command in a cmdtest Session", func() {
		session := runner.Run("bash", "-c", "echo hi out; echo hi err 1>&2; exit 42")

		Expect(session).To(Say("hi out"))
		Expect(session).To(SayError("hi err"))
		Expect(session).To(ExitWith(42))
	})
})

var _ = Describe("Curl", func() {
	It("outputs the body of the given URL", func() {
		session := &cmdtest.Session{}

		runner.SessionStarter = func(cmd *exec.Cmd) (*cmdtest.Session, error) {
			Expect(cmd.Path).To(Equal(exec.Command("curl").Path))

			Expect(cmd.Args).To(Equal([]string{
				"curl", "-s", "http://example.com",
			}))

			return session, nil
		}

		var someSession *cmdtest.Session
		someSession = runner.Curl("http://example.com")

		Expect(someSession).To(Equal(session))
	})
})
