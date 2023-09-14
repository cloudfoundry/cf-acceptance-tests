package windows

import (
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/helpers"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/logs"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/skip_messages"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = WindowsDescribe("Readiness Healthcheck", func() {
	var appName string
	var readinessHealthCheckTimeout = "25s" // 20s route emitter sync loop + 2s hc interval + bonus

	BeforeEach(func() {
		if !Config.GetReadinessHealthChecksEnabled() {
			Skip(skip_messages.SkipReadinessHealthChecksMessage)
		}
		appName = random_name.CATSRandomName("APP")
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)
		Eventually(cf.Cf("delete", appName, "-f")).Should(Exit(0))
	})

	Describe("when the readiness healthcheck is set to http", func() {
		It("registers the route only when the readiness check passes", func() {
			By("pushing the app")
			Expect(cf.Cf("push",
				appName,
				"-f", assets.NewAssets().Nora+"/../readiness_manifest.yml",
				"-p", assets.NewAssets().Nora,
				"-s", Config.GetWindowsStack(),
			).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			By("verifying the app starts")
			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}, Config.CfPushTimeoutDuration()).Should(ContainSubstring("hello i am nora running on"))

			By("verifying the app is marked as ready")
			Eventually(func() string {
				return helpers.CurlApp(Config, appName, "/ready")
			}, Config.DefaultTimeoutDuration()).Should(ContainSubstring("200 - ready"))

			// TODO: only include this when audit events are built
			// Eventually(cf.Cf("events", appName)).Should(Say("app.ready"))

			Expect(logs.Recent(appName).Wait()).To(Say("Container passed the readiness health check"))

			By("triggering the app to make the /ready endpoint fail")
			helpers.CurlApp(Config, appName, "/ready/false")

			By("verifying the app is marked as not ready")

			// TODO: only include this when audit events are built
			// Eventually(cf.Cf("events", appName)).Should(Say("app.notready"))

			Eventually(func() BufferProvider { return logs.Recent(appName).Wait() }, readinessHealthCheckTimeout).Should(Say("Container failed the readiness health check"))

			By("verifying the app is removed from the routing table")
			Eventually(func() string {
				return helpers.CurlApp(Config, appName, "/ready")
			}, readinessHealthCheckTimeout).Should(ContainSubstring("404"))

			By("verifying that the app hasn't restarted")
			Consistently(cf.Cf("events", appName)).ShouldNot(Say("audit.app.process.crash"))

			if Config.GetIncludeSsh() {
				By("re-enabling the app's readiness endpoint")
				Expect(cf.Cf("ssh", appName, "-c", "curl localhost:8080/ready/true").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))

				By("verifying the app is re-added to the routing table")
				Eventually(func() string {
					return helpers.CurlApp(Config, appName, "/ready")
				}, readinessHealthCheckTimeout).Should(ContainSubstring("200 - ready"))

				By("verifying the app has not restarted")
				Consistently(cf.Cf("events", appName)).ShouldNot(Say("audit.app.process.crash"))
			}
		})
	})
})
