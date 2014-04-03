package apps

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/vito/cmdtest/matchers"

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
			Expect(Cf("push", appName, "-p", NewAssets().Node, "-c", "node app.js")).To(Say("App started"))

			Eventually(Curling(appName, "/", LoadConfig().AppsDomain)).Should(Say("Hello from a node app!"))

			Expect(Cf("delete", appName, "-f")).To(Say("OK"))
		})
	})

	Describe("java", func() {
		It("makes the app reachable via its bound route", func() {
			Expect(Cf("push", appName, "-p", NewAssets().Java)).To(Say("App started"))

			Eventually(Curling(appName, "/", LoadConfig().AppsDomain)).Should(Say("Hello, from your friendly neighborhood Java JSP!"))

			Expect(Cf("delete", appName, "-f")).To(Say("OK"))
		})
	})
})
