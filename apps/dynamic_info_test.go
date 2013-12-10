package apps

import (
	"github.com/vito/cmdtest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/vito/cmdtest/matchers"

	. "github.com/pivotal-cf-experimental/cf-acceptance-tests/helpers"
)

var _ = Describe("A running application", func() {
	BeforeEach(func() {
		AppName = RandomName()

		Expect(Cf("push", AppName, "-p", doraPath)).To(Say("Started"))
	})

	AfterEach(func() {
		Expect(Cf("delete", AppName, "-f")).To(Say("OK"))
	})

	It("can have its files inspected", func() {
		// Currently cannot work with multiple instances since GCF always checks instance 0
		Expect(Cf("files", AppName)).To(Say("app/"))
		Expect(Cf("files", AppName, "app/")).To(Say("config.ru"))
		Expect(Cf("files", AppName, "app/config.ru")).To(
			Say("run Dora"),
		)
	})

	It("can show crash events", func() {
		Expect(Curl(AppUri("/sigterm/KILL"))).To(ExitWith(0))
		Eventually(func() *cmdtest.Session {
			return Cf("events", AppName)
		}, 10).Should(Say("exited"))
	})
	
	Context("with multiple instances", func() {
		BeforeEach(func() {
			Expect(
				Cf("scale", AppName, "-i", "2"),
			).To(Say("OK"))
		})

		It("can be queried for state by instance", func() {
			app := Cf("app", AppName)
			Expect(app).To(Say("#0"))
			Expect(app).To(Say("#1"))
		})
	})
})
