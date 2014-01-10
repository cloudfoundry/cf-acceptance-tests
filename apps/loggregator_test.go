package apps

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/vito/cmdtest/matchers"

	. "github.com/pivotal-cf-experimental/cf-acceptance-tests/helpers"
)

var _ = Describe("loggregator", func() {
	BeforeEach(func() {
		AppName = RandomName()

		PushApp(AppName, doraPath)
	})

	AfterEach(func() {
		DeleteApp(AppName)
	})

	Context("gcf logs", func() {
		It("blocks and exercises basic loggregator behavior", func() {
			logs := Cf("logs", AppName)

			Eventually(Curling("/")).Should(Say("Hi, I'm Dora!"))

			Expect(logs).To(SayWithTimeout("OUT "+AppName+"."+IntegrationConfig.AppsDomain, time.Second*15))
		})
	})

	Context("gcf logs --recent", func() {
		It("makes loggregator buffer and dump log messages", func() {
			logs := Cf("logs", AppName, "--recent")

			Expect(logs).To(Say("OUT Starting app instance"))

			Eventually(Curling("/")).Should(Say("Hi, I'm Dora!"))

			Expect(logs).ToNot(Say("OUT " + AppName + "." + IntegrationConfig.AppsDomain))
		})
	})
})
