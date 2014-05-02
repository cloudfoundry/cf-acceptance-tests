package runner_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
	"github.com/pivotal-cf-experimental/cf-test-helpers/runner"
)

var _ = Describe("Run", func() {
	It("runs the given command in a cmdtest Session", func() {
		session := runner.Run("bash", "-c", "echo hi out; echo hi err 1>&2; exit 42")
		Eventually(session.Out).Should(Say("hi out"))
		Eventually(session.Err).Should(Say("hi err"))
		Eventually(session).Should(Exit(42))
	})
})

var _ = Describe("Curl", func() {
	It("outputs the body of the given URL", func() {
		session := runner.Curl("-I", "http://example.com")

		Eventually(session.Out).Should(Say("HTTP/1.1 200 OK"))
	})
})
