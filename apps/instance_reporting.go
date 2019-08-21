package apps

import (
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = AppsDescribe("Getting instance information", func() {
	Describe("scaling memory", func() {
		var appName string
		var runawayTestSetup *workflowhelpers.ReproducibleTestSuiteSetup

		BeforeEach(func() {
			runawayTestSetup = workflowhelpers.NewRunawayAppTestSuiteSetup(Config)
			runawayTestSetup.Setup()

			appName = random_name.CATSRandomName("APP")

			Eventually(cf.Cf(
				"push", appName,
				"-p", assets.NewAssets().Binary,
				"-b", "binary_buildpack",
				"-m", DEFAULT_MEMORY_LIMIT,
				"-c", "./app"),
				Config.CfPushTimeoutDuration()).Should(Exit(0))
		})

		AfterEach(func() {
			app_helpers.AppReport(appName)
			Eventually(cf.Cf("delete", appName, "-f")).Should(Exit(0))

			runawayTestSetup.Teardown()
		})

		PIt("fails with insufficient resources", func() {
			scale := cf.Cf("scale", appName, "-m", workflowhelpers.RUNAWAY_QUOTA_MEM_LIMIT, "-f")
			Eventually(scale).Should(Or(Say("insufficient"), Say("down")))
			scale.Kill()

			app := cf.Cf("app", appName)
			Eventually(app).Should(Exit(0))
			Expect(app.Out).NotTo(Say("instances: 1/1"))
		})
	})

	Describe("Scaling instances", func() {
		var appName string

		BeforeEach(func() {
			appName = random_name.CATSRandomName("APP")

			Expect(cf.Cf("push",
				appName,
				"-b", Config.GetBinaryBuildpackName(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", assets.NewAssets().Catnip,
				"-c", "./catnip",
				"-i", "1",
			).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		})

		AfterEach(func() {
			app_helpers.AppReport(appName)

			Expect(cf.Cf("delete", appName, "-f", "-r").Wait()).To(Exit(0))
		})

		It("can be queried for state by instance", func() {
			Expect(cf.Cf("scale", appName, "-i", "2").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			app := cf.Cf("app", appName).Wait()
			Expect(app).To(Exit(0))
			Expect(app).To(Say("#0"))
			Expect(app).To(Say("#1"))
		})
	})
})
