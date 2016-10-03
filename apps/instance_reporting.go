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
				"--no-start",
				"-b", "binary_buildpack",
				"-m", DEFAULT_MEMORY_LIMIT,
				"-d", Config.GetAppsDomain(),
				"-c", "./app"),
				Config.CfPushTimeoutDuration()).Should(Exit(0))

			app_helpers.SetBackend(appName)
			Eventually(cf.Cf("start", appName), Config.CfPushTimeoutDuration()).Should(Exit(0))
		})

		AfterEach(func() {
			app_helpers.AppReport(appName, Config.DefaultTimeoutDuration())
			Eventually(cf.Cf("delete", appName, "-f"), Config.DefaultTimeoutDuration()).Should(Exit(0))

			runawayTestSetup.Teardown()
		})

		It("fails with insufficient resources", func() {
			scale := cf.Cf("scale", appName, "-m", workflowhelpers.RUNAWAY_QUOTA_MEM_LIMIT, "-f")
			Eventually(scale, Config.DefaultTimeoutDuration()).Should(Or(Say("insufficient"), Say("down")))
			scale.Kill()

			app := cf.Cf("app", appName)
			Eventually(app, Config.DefaultTimeoutDuration()).Should(Exit(0))
			Expect(app.Out).NotTo(Say("instances: 1/1"))
		})
	})
})
