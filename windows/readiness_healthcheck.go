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

	// TODO add a ready endpoint to Nora
	Describe("when the readiness healthcheck is set to http", func() {
		It("registers the route only when the readiness check passes", func() {
			By("pushing the app")
			// FIXME use manifest and update linux
			Expect(cf.Cf("push",
				appName,
				"-f", assets.NewAssets().Nora+"/../readiness_manifest.yml",
				"-p", assets.NewAssets().Nora,
			).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			By("verifying the app starts")
			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}, Config.CfPushTimeoutDuration()).Should(ContainSubstring("hello i am nora running on"))

			By("verifying the app is marked as ready")
			// TODO: only include this when audit events are built
			// Eventually(func() string {
			// 	return string(cf.Cf("events", appName).Wait().Out.Contents())
			// }).Should(MatchRegexp("app.ready"))

			// Windows apps take a Loooooong time to start apparently
			Expect(string(logs.Recent(appName).Wait(Config.CfPushTimeoutDuration()).Out.Contents())).Should(ContainSubstring("Container passed the readiness health check"))

			By("triggering the app to make the /ready endpoint fail")
			helpers.CurlApp(Config, appName, "/ready/false")

			By("verifying the app is marked as not ready")

			// TODO: only include this when audit events are built
			// Eventually(func() string {
			// 	return string(cf.Cf("events", appName).Wait().Out.Contents())
			// }).Should(MatchRegexp("app.notready"))

			Eventually(func() string {
				return string(logs.Recent(appName).Wait().Out.Contents())
			}, readinessHealthCheckTimeout).Should(ContainSubstring("Container failed the readiness health check"))

			By("verifying the app is removed from the routing table")
			Eventually(func() string {
				return helpers.CurlApp(Config, appName, "/ready")
			}).Should(ContainSubstring("404 Not Found"))

			By("verifying that the app hasn't restarted")
			Consistently(func() string {
				return string(cf.Cf("events", appName).Wait().Out.Contents())
			}).ShouldNot(MatchRegexp("audit.app.process.crash"))

			By("re-enabling the app's readiness endpoint")
			Expect(cf.Cf("ssh", appName, "-c", "curl localhost:8080/ready/true").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))

			By("verifying the app is re-added to the routing table")
			Eventually(func() string {
				return helpers.CurlApp(Config, appName, "/ready")
			}, readinessHealthCheckTimeout).Should(ContainSubstring("200 - ready"))

			By("verifying the app has not restarted")
			Consistently(func() string {
				return string(cf.Cf("events", appName).Wait().Out.Contents())
			}).ShouldNot(MatchRegexp("audit.app.process.crash"))

		})
	})
})
