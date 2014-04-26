package apps

import (
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	. "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/generator"
)

var _ = PDescribe("loggregator", func() {
	var appName string

	BeforeEach(func() {
		appName = RandomName()

		Eventually(Cf("push", appName, "-p", NewAssets().Dora), CFPushTimeout).Should(Exit(0))
	})

	AfterEach(func() {
		Eventually(Cf("delete", appName, "-f"), DefaultTimeout).Should(Exit(0))
	})

	Context("gcf logs", func() {
		It("blocks and exercises basic loggregator behavior", func() {
			logs := Cf("logs", appName)

			Eventually(logs, DefaultTimeout).Should(Say("Connected, tailing logs for app"))

			Eventually(CurlFetcher(appName, "/", LoadConfig().AppsDomain), DefaultTimeout).Should(ContainSubstring("Hi, I'm Dora!"))

			Eventually(logs, DefaultTimeout).Should(Say("OUT " + appName + "." + LoadConfig().AppsDomain))
		})
	})

	Context("gcf logs --recent", func() {
		It("makes loggregator buffer and dump log messages", func() {
			logs := Cf("logs", appName, "--recent")

			Eventually(logs, DefaultTimeout).Should(Say("Connected, dumping recent logs for app"))
			Eventually(logs, DefaultTimeout).Should(Say("OUT Created app"))
			Eventually(logs, DefaultTimeout).Should(Say("OUT Starting app instance"))
		})
	})
})
