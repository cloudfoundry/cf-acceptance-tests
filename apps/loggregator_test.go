package apps

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/vito/cmdtest/matchers"

    . "github.com/pivotal-cf-experimental/cf-test-helpers/generator"
    . "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
	. "github.com/pivotal-cf-experimental/cf-acceptance-tests/helpers"
	"time"
)

var _ = PDescribe("loggregator", func() {
	BeforeEach(func() {
		AppName = RandomName()

		PushApp(AppName, doraPath)
	})

	AfterEach(func() {
		DeleteApp(AppName)
	})

	Context("gcf logs", func() {
		PIt("blocks and exercises basic loggregator behavior", func() {
			logs := Cf("logs", AppName)

			Expect(logs).To(SayWithTimeout("Connected, tailing logs for app", time.Second*15))

			Eventually(Curling("/")).Should(Say("Hi, I'm Dora!"))

			Expect(logs).To(SayWithTimeout("OUT "+AppName+"."+IntegrationConfig.AppsDomain, time.Second*15))
		})
	})

	Context("gcf logs --recent", func() {
		It("makes loggregator buffer and dump log messages", func() {
		   	logs := Cf("logs", AppName, "--recent")

			Expect(logs).To(SayWithTimeout("Connected, dumping recent logs for app", time.Second*15))

			Expect(logs).To(SayWithTimeout("OUT Created app", time.Second*15))
			Expect(logs).To(SayWithTimeout("OUT Starting app instance", time.Second*15))
		})
	})
})
