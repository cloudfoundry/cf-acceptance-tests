package apps

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	. "github.com/cloudfoundry/cf-acceptance-tests/helpers"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/generator"
)

var _ = Describe("Application", func() {
	var appName string

	BeforeEach(func() {
		appName = RandomName()

		Eventually(Cf("push", appName, "-p", NewAssets().Dora), CFPushTimeout).Should(Exit(0))
	})

	AfterEach(func() {
		Eventually(Cf("delete", appName, "-f"), DefaultTimeout).Should(Exit(0))
	})

	Describe("pushing", func() {
		It("makes the app reachable via its bound route", func() {
			Eventually(CurlFetcher(appName, "/", LoadConfig().AppsDomain), DefaultTimeout).Should(ContainSubstring("Hi, I'm Dora!"))
		})
	})

	Describe("stopping", func() {
		BeforeEach(func() {
			Eventually(Cf("stop", appName), DefaultTimeout).Should(Exit(0))
		})

		It("makes the app unreachable", func() {
			Eventually(CurlFetcher(appName, "/", LoadConfig().AppsDomain), DefaultTimeout).Should(ContainSubstring("404"))
		})

		Describe("and then starting", func() {
			BeforeEach(func() {
				Eventually(Cf("start", appName), DefaultTimeout).Should(Exit(0))
			})

			It("makes the app reachable again", func() {
				Eventually(CurlFetcher(appName, "/", LoadConfig().AppsDomain), DefaultTimeout).Should(ContainSubstring("Hi, I'm Dora!"))
			})
		})
	})

	Describe("updating", func() {
		It("is reflected through another push", func() {
			Eventually(CurlFetcher(appName, "/", LoadConfig().AppsDomain), DefaultTimeout).Should(ContainSubstring("Hi, I'm Dora!"))

			Eventually(Cf("push", appName, "-p", NewAssets().HelloWorld), CFPushTimeout).Should(Exit(0))

			Eventually(CurlFetcher(appName, "/", LoadConfig().AppsDomain), DefaultTimeout).Should(ContainSubstring("Hello, world!"))
		})
	})

	Describe("deleting", func() {
		BeforeEach(func() {
			Eventually(Cf("delete", appName, "-f"), DefaultTimeout).Should(Exit(0))
		})

		It("removes the application", func() {
			Eventually(Cf("app", appName), DefaultTimeout).Should(Say("not found"))
		})

		It("makes the app unreachable", func() {
			Eventually(CurlFetcher(appName, "/", LoadConfig().AppsDomain), DefaultTimeout).Should(ContainSubstring("404"))
		})
	})
})
