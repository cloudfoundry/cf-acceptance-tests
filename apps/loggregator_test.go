package apps

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/vito/cmdtest/matchers"

	. "github.com/pivotal-cf-experimental/cf-acceptance-tests/helpers"
)

var _ = Describe("gcf logs <app-name>", func() {
	BeforeEach(func() {
		AppName = RandomName()

		PushApp(AppName, doraPath)
	})

	AfterEach(func() {
		DeleteApp(AppName)
	})

	Context("by default", func() {
		It("contains an app registration log message with console colors", func() {
			logs := Cf("logs", AppName)

			Eventually(Curling("/")).Should(Say("Hi, I'm Dora!"))

			Expect(logs).To(SayWithTimeout("\\[DEA\\]\\x1b\\[0m     \\x1b\\[0;38mOUT Registering app instance",
									time.Second * 10))
		})
	})

	Context("--recent", func() {
		It("contains recent app log messages with console colors", func() {
			Eventually(Curling("/")).Should(Say("Hi, I'm Dora!"))

			logs := Cf("logs", AppName, "--recent")
			Expect(logs).To(Say("\\[DEA\\]\\x1b\\[0m     \\x1b\\[0;38mOUT Registering app instance"))
			Expect(logs).To(Say("\\[RTR\\]\\x1b\\[0m     \\x1b\\[0;38mOUT " + AppName + "." + IntegrationConfig.AppsDomain))
		})
	})
})
