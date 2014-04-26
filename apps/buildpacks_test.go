package apps

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	. "github.com/cloudfoundry/cf-acceptance-tests/helpers"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/generator"
)

var _ = Describe("Application", func() {
	var appName string

	BeforeEach(func() {
		appName = RandomName()
	})

	Describe("node", func() {
		It("makes the app reachable via its bound route", func() {
			Eventually(Cf("push", appName, "-p", NewAssets().Node, "-c", "node app.js"), CFPushTimeout).Should(Exit(0))

			Eventually(CurlFetcher(appName, "/", LoadConfig().AppsDomain), DefaultTimeout).Should(ContainSubstring("Hello from a node app!"))

			Eventually(Cf("delete", appName, "-f"), DefaultTimeout).Should(Exit(0))
		})
	})

	Describe("java", func() {
		It("makes the app reachable via its bound route", func() {
			Eventually(Cf("push", appName, "-p", NewAssets().Java), CFPushTimeout).Should(Exit(0))

			Eventually(CurlFetcher(appName, "/", LoadConfig().AppsDomain), DefaultTimeout).Should(ContainSubstring("Hello, from your friendly neighborhood Java JSP!"))

			Eventually(Cf("delete", appName, "-f"), DefaultTimeout).Should(Exit(0))
		})
	})

	Describe("go", func() {
		It("makes the app reachable via its bound route", func() {
			Eventually(Cf("push", appName, "-p", NewAssets().Go), CFPushTimeout).Should(Exit(0))

			Eventually(CurlFetcher(appName, "/", LoadConfig().AppsDomain), DefaultTimeout).Should(ContainSubstring("go, world"))

			Eventually(Cf("delete", appName, "-f"), DefaultTimeout).Should(Exit(0))
		})
	})
})
