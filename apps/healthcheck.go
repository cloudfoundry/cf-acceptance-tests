package apps

import (
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/logs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"path/filepath"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
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
				"-p", assets.NewAssets().WorkerApp,
				"-f", filepath.Join(assets.NewAssets().WorkerApp, "manifest.yml"),
				"-b", "go_buildpack",
				"-m", DEFAULT_MEMORY_LIMIT,
				"-i", "1",
				"-u", "process"),
				Config.CfPushTimeoutDuration(),
			).Should(Exit(0))

			By("verifying it's up")
			Eventually(func() *Session {
				appLogsSession := logs.Tail(Config.GetUseLogCache(), appName)
				Expect(appLogsSession.Wait()).To(Exit(0))
				return appLogsSession
			}).Should(gbytes.Say("I am working at"))
		})
	})

	Describe("when the healthcheck is set to port", func() {
		It("starts up successfully", func() {
			By("pushing it")
			Eventually(cf.Cf(
				"push", appName,
				"-b", Config.GetBinaryBuildpackName(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", assets.NewAssets().Catnip,
				"-c", "./catnip",
				"-i", "1",
				"-u", "port"),
				Config.CfPushTimeoutDuration(),
			).Should(Exit(0))

			By("verifying it's up")
			Eventually(helpers.CurlingAppRoot(Config, appName)).Should(ContainSubstring("Catnip?"))
		})
	})

	Describe("when the healthcheck is set to http", func() {
		It("starts up successfully", func() {
			By("pushing it")
			Eventually(cf.Cf(
				"push", appName,
				"-b", Config.GetBinaryBuildpackName(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", assets.NewAssets().Catnip,
				"-c", "./catnip",
				"-i", "1",
				"-u", "port"),
				Config.CfPushTimeoutDuration(),
			).Should(Exit(0))

			cf.Cf("curl", appName, "-X", "PUT", "-d", `{"HealthCheckType":"http", "HealthCheckHTTPEndpoint":"/health"}`)

			By("verifying it's up")
			Eventually(helpers.CurlingAppRoot(Config, appName)).Should(ContainSubstring("Catnip?"))
		})
	})
})
