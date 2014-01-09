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
		It("contains a router message with console colors from visiting the app", func() {
			logs := Cf("logs", AppName)

			Eventually(Curling("/")).Should(Say("Hi, I'm Dora!"))

			Expect(logs).To(SayWithTimeout("\\[RTR\\]\\x1b\\[0m     \\x1b\\[0;38mOUT "+AppName+"."+IntegrationConfig.AppsDomain,
				time.Second*15))
		})
	})

	Context("--recent", func() {
		It("contains recent app log messages with console colors", func() {
			logs := Cf("logs", AppName, "--recent")

			Expect(logs).To(Say("\\[DEA\\]\\x1b\\[0m     \\x1b\\[0;38mOUT Starting app instance"))

			Eventually(Curling("/")).Should(Say("Hi, I'm Dora!"))

			Expect(logs).ToNot(Say("\\[RTR\\]\\x1b\\[0m     \\x1b\\[0;38mOUT " + AppName + "." + IntegrationConfig.AppsDomain))
		})
	})
})
