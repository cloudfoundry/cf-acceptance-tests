package diego

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	. "github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers"
)

var _ = Describe("Application staging with Diego", func() {
	var appName string

	BeforeEach(func() {
		appName = RandomName()

		//Diego needs a custom buildpack until the ruby buildpack lands
		Eventually(Cf("push", appName, "-p", NewAssets().Dora, "--no-start", "-b=https://github.com/cloudfoundry/cf-buildpack-ruby/archive/master.zip"), CFPushTimeout).Should(Exit(0))
		Eventually(Cf("set-env", appName, "CF_DIEGO_BETA", "true"), DefaultTimeout).Should(Exit(0))
		Eventually(Cf("start", appName), CFPushTimeout).Should(Exit(0))
	})

	AfterEach(func() {
		Eventually(Cf("delete", appName, "-f"), DefaultTimeout).Should(Exit(0))
	})

	Describe("pushing", func() {
		It("makes the app reachable via its bound route", func() {
			Expect(CurlAppRoot(appName)).To(ContainSubstring("Hi, I'm Dora!"))
		})
	})

	Describe("stopping", func() {
		BeforeEach(func() {
			Eventually(Cf("stop", appName), DefaultTimeout).Should(Exit(0))
		})

		It("makes the app unreachable", func() {
			Expect(CurlAppRoot(appName)).To(ContainSubstring("404"))
		})

		Describe("and then starting", func() {
			BeforeEach(func() {
				Eventually(Cf("start", appName), DefaultTimeout).Should(Exit(0))
			})

			It("makes the app reachable again", func() {
				Expect(CurlAppRoot(appName)).To(ContainSubstring("Hi, I'm Dora!"))
			})
		})
	})

	Describe("updating", func() {
		It("is reflected through another push", func() {
			Expect(CurlAppRoot(appName)).To(ContainSubstring("Hi, I'm Dora!"))

			Eventually(Cf("push", appName, "-p", NewAssets().HelloWorld), CFPushTimeout).Should(Exit(0))

			Expect(CurlAppRoot(appName)).To(ContainSubstring("Hello, world!"))
		})
	})

	Describe("deleting", func() {
		BeforeEach(func() {
			Eventually(Cf("delete", appName, "-f"), DefaultTimeout).Should(Exit(0))
		})

		It("removes the application and makes the app unreachable", func() {
			Eventually(Cf("app", appName), DefaultTimeout).Should(Say("not found"))
			Expect(CurlAppRoot(appName)).To(ContainSubstring("404"))
		})
	})
})
