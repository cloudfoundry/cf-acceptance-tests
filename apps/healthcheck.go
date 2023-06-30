package apps

import (
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/logs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/helpers"
)

var _ = AppsDescribe("Healthcheck", func() {
	var appName string

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)
		Eventually(cf.Cf("delete", appName, "-f")).Should(Exit(0))
	})

	Describe("when the healthcheck is set to none", func() {
		It("starts up successfully", func() {
			By("pushing it")
			Eventually(cf.Cf(
				"push", appName,
				"-p", assets.NewAssets().Worker,
				"-b", Config.GetGoBuildpackName(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-i", "1",
				"-u", "process"),
				Config.CfPushTimeoutDuration(),
			).Should(Exit(0))

			By("verifying it's up")
			Eventually(func() *Session {
				appLogsSession := logs.Recent(appName)
				Expect(appLogsSession.Wait()).To(Exit(0))
				return appLogsSession
			}).Should(gbytes.Say("Running Worker"))
		})
	})

	Describe("when the healthcheck is set to port", func() {
		It("starts up successfully", func() {
			By("pushing it")

			Eventually(cf.Cf(app_helpers.CatnipWithArgs(appName, "-m", DEFAULT_MEMORY_LIMIT, "-i", "1", "-u", "port")...),
				Config.CfPushTimeoutDuration(),
			).Should(Exit(0))

			By("verifying it's up")
			Eventually(helpers.CurlingAppRoot(Config, appName)).Should(ContainSubstring("Catnip?"))
		})
	})

	Describe("when the healthcheck is set to http", func() {
		It("starts up successfully", func() {
			By("pushing it")

			Eventually(cf.Cf(app_helpers.CatnipWithArgs(appName, "-m", DEFAULT_MEMORY_LIMIT, "-i", "1", "-u", "port")...),
				Config.CfPushTimeoutDuration(),
			).Should(Exit(0))

			cf.Cf("curl", appName, "-X", "PUT", "-d", `{"HealthCheckType":"http", "HealthCheckHTTPEndpoint":"/health"}`)

			By("verifying it's up")
			Eventually(helpers.CurlingAppRoot(Config, appName)).Should(ContainSubstring("Catnip?"))
		})
	})
})
