package apps

import (
	"github.com/vito/cmdtest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/vito/cmdtest/matchers"

	. "github.com/cloudfoundry/cf-acceptance-tests/helpers"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/generator"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/runner"
)

var _ = Describe("A running application", func() {
	var appName string

	BeforeEach(func() {
		appName = RandomName()

		Expect(Cf("push", appName, "-p", NewAssets().Dora)).To(Say("App started"))
	})

	AfterEach(func() {
		Expect(Cf("delete", appName, "-f")).To(Say("OK"))
	})

	It("can have its files inspected", func() {
		// Currently cannot work with multiple instances since GCF always checks instance 0
		Expect(Cf("files", appName)).To(Say("app/"))
		Expect(Cf("files", appName, "app/")).To(Say("config.ru"))
		Expect(Cf("files", appName, "app/config.ru")).To(
			Say("run Dora"),
		)
	})

	It("can show crash events", func() {
		Expect(Curl(AppUri(appName, "/sigterm/KILL", LoadConfig().AppsDomain))).To(ExitWith(0))
		Eventually(func() *cmdtest.Session {
			return Cf("events", appName)
		}, 10).Should(Say("exited"))
	})

	Context("with multiple instances", func() {
		BeforeEach(func() {
			Expect(
				Cf("scale", appName, "-i", "2"),
			).To(Say("OK"))
		})

		It("can be queried for state by instance", func() {
			app := Cf("app", appName)
			Expect(app).To(Say("#0"))
			Expect(app).To(Say("#1"))
		})
	})
})
