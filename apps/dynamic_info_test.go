package apps

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	. "github.com/cloudfoundry/cf-acceptance-tests/helpers"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/generator"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/runner"
)

var _ = Describe("A running application", func() {
	var appName string

	BeforeEach(func() {
		appName = RandomName()

		Eventually(Cf("push", appName, "-p", NewAssets().Dora), CFPushTimeout).Should(Exit(0))
	})

	AfterEach(func() {
		Eventually(Cf("delete", appName, "-f"), DefaultTimeout).Should(Exit(0))
	})

	It("can have its files inspected", func() {
		// Currently cannot work with multiple instances since GCF always checks instance 0
		Eventually(Cf("files", appName), DefaultTimeout).Should(Say("app/"))
		Eventually(Cf("files", appName, "app/"), DefaultTimeout).Should(Say("config.ru"))
		Eventually(Cf("files", appName, "app/config.ru"), DefaultTimeout).Should(Say("run Dora"))
	})

	It("can show crash events", func() {
		Eventually(Curl(AppUri(appName, "/sigterm/KILL", LoadConfig().AppsDomain)), DefaultTimeout).Should(Exit(0))
		Eventually(func() string {
			return string(Cf("events", appName).Wait(DefaultTimeout).Out.Contents())
		}, DefaultTimeout).Should(ContainSubstring("exited"))
	})

	Context("with multiple instances", func() {
		BeforeEach(func() {
			Eventually(Cf("scale", appName, "-i", "2"), DefaultTimeout).Should(Exit(0))
		})

		It("can be queried for state by instance", func() {
			app := Cf("app", appName)
			Eventually(app, DefaultTimeout).Should(Say("#0"))
			Eventually(app, DefaultTimeout).Should(Say("#1"))
		})
	})
})
