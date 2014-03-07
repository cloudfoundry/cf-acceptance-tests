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

		Expect(Cf("push", appName, "-p", NewAssets().Dora)).To(Say("App started"))
	})

	AfterEach(func() {
		Expect(Cf("delete", appName, "-f")).To(Say("OK"))
	})

	Describe("pushing", func() {
		It("makes the app reachable via its bound route", func() {
			Eventually(Curling(appName, "/", LoadConfig().AppsDomain)).Should(Say("Hi, I'm Dora!"))
		})
	})

	Describe("stopping", func() {
		BeforeEach(func() {
			Expect(Cf("stop", appName)).To(Say("OK"))
		})

		It("makes the app unreachable", func() {
			Eventually(Curling(appName, "/", LoadConfig().AppsDomain), 5.0).Should(Say("404"))
		})

		Describe("and then starting", func() {
			BeforeEach(func() {
				Expect(Cf("start", appName)).To(Say("App started"))
			})

			It("makes the app reachable again", func() {
				Eventually(Curling(appName, "/", LoadConfig().AppsDomain)).Should(Say("Hi, I'm Dora!"))
			})
		})
	})

	Describe("updating", func() {
		It("is reflected through another push", func() {
			Eventually(Curling(appName, "/", LoadConfig().AppsDomain)).Should(Say("Hi, I'm Dora!"))

			Expect(Cf("push", appName, "-p", NewAssets().HelloWorld)).To(Say("App started"))

			Eventually(Curling(appName, "/", LoadConfig().AppsDomain)).Should(Say("Hello, world!"))
		})
	})

	Describe("deleting", func() {
		BeforeEach(func() {
			Expect(Cf("delete", appName, "-f")).To(Say("OK"))
		})

		It("removes the application", func() {
			Expect(Cf("app", appName)).To(Say("not found"))
		})

		It("makes the app unreachable", func() {
			Eventually(Curling(appName, "/", LoadConfig().AppsDomain)).Should(Say("404"))
		})
	})
})
