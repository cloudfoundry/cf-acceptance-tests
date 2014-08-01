package diego

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers"
)

var _ = Describe("Application Lifecycle", func() {
	var appName string

	describeLifeCycle := func() {
		Describe("pushing", func() {
			It("makes the app reachable via its bound route", func() {
				Eventually(helpers.CurlingAppRoot(appName)).Should(ContainSubstring("Hi, I'm Dora!"))
			})
		})

		Describe("stopping", func() {
			BeforeEach(func() {
				Eventually(cf.Cf("stop", appName), DEFAULT_TIMEOUT).Should(Exit(0))
			})

			It("makes the app unreachable", func() {
				Eventually(helpers.CurlingAppRoot(appName)).Should(ContainSubstring("404"))
			})

			Describe("and then starting", func() {
				BeforeEach(func() {
					Eventually(cf.Cf("start", appName), DEFAULT_TIMEOUT).Should(Exit(0))
				})

				It("makes the app reachable again", func() {
					Eventually(helpers.CurlingAppRoot(appName)).Should(ContainSubstring("Hi, I'm Dora!"))
				})
			})
		})

		Describe("updating", func() {
			It("is reflected through another push", func() {
				Eventually(helpers.CurlingAppRoot(appName)).Should(ContainSubstring("Hi, I'm Dora!"))

				Eventually(cf.Cf("push", appName, "-p", helpers.NewAssets().HelloWorld), CF_PUSH_TIMEOUT).Should(Exit(0))

				Eventually(helpers.CurlingAppRoot(appName)).Should(ContainSubstring("Hello, world!"))
			})
		})

		Describe("deleting", func() {
			BeforeEach(func() {
				Eventually(cf.Cf("delete", appName, "-f"), DEFAULT_TIMEOUT).Should(Exit(0))
			})

			It("removes the application and makes the app unreachable", func() {
				Eventually(cf.Cf("app", appName), DEFAULT_TIMEOUT).Should(Say("not found"))
				Eventually(helpers.CurlingAppRoot(appName)).Should(ContainSubstring("404"))
			})
		})
	}

	Describe("An app staged with Diego and running on a DEA", func() {
		BeforeEach(func() {
			appName = generator.RandomName()

			Eventually(cf.Cf("push", appName, "-p", helpers.NewAssets().Dora, "--no-start"), CF_PUSH_TIMEOUT).Should(Exit(0))
			Eventually(cf.Cf("set-env", appName, "CF_DIEGO_BETA", "true"), DEFAULT_TIMEOUT).Should(Exit(0))
			Eventually(cf.Cf("start", appName), CF_PUSH_TIMEOUT).Should(Exit(0))
		})

		AfterEach(func() {
			Eventually(cf.Cf("delete", appName, "-f"), DEFAULT_TIMEOUT).Should(Exit(0))
		})

		describeLifeCycle()
	})

	Describe("An app both staged and run with Diego", func() {
		BeforeEach(func() {
			appName = generator.RandomName()

			Eventually(cf.Cf("push", appName, "-p", helpers.NewAssets().Dora, "--no-start"), CF_PUSH_TIMEOUT).Should(Exit(0))
			Eventually(cf.Cf("set-env", appName, "CF_DIEGO_RUN_BETA", "true"), DEFAULT_TIMEOUT).Should(Exit(0)) // CF_DIEGO_RUN_BETA also implies CF_DIEGO_BETA in CC
			Eventually(cf.Cf("start", appName), CF_PUSH_TIMEOUT).Should(Exit(0))
		})

		AfterEach(func() {
			Eventually(cf.Cf("delete", appName, "-f"), DEFAULT_TIMEOUT).Should(Exit(0))
		})

		describeLifeCycle()
	})
})
