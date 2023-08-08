package apps

import (
	"fmt"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/logs"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/skip_messages"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/helpers"
	"github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers"
)

var _ = AppsDescribe("Readiness Healthcheck", func() {
	var appName, proxyAppName string
	var orgName string
	var spaceName string
	var readinessHealthCheckTimeout = "25s" // 20s route emitter sync loop + 2s hc interval + bonus

	BeforeEach(func() {
		if !Config.GetReadinessHealthChecksEnabled() {
			Skip(skip_messages.SkipReadinessHealthChecksMessage)
		}
		appName = random_name.CATSRandomName("APP")
		proxyAppName = random_name.CATSRandomName("APP")
		orgName = TestSetup.RegularUserContext().Org
		spaceName = TestSetup.RegularUserContext().Space
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)
		app_helpers.AppReport(proxyAppName)
		Eventually(cf.Cf("delete", appName, "-f")).Should(Exit(0))
		Eventually(cf.Cf("delete", proxyAppName, "-f")).Should(Exit(0))
	})

	Describe("when the readiness healthcheck is set to http", func() {
		FIt("registers the route only when the readiness check passes", func() {
			By("pushing the app")
			Expect(cf.Push(appName,
				"-p", assets.NewAssets().Dora,
				"--readiness-endpoint", "/ready",
				"--readiness-health-check-type", "http",
				"--readiness-health-check-interval", "1",
			).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			By("verifying the app starts")
			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}, Config.DefaultTimeoutDuration()).Should(ContainSubstring("Hi, I'm Dora!"))

			// Get the initial overlay IP for dora app
			overlayIP := helpers.CurlApp(Config, appName, "/myip")

			By("verifying the app is marked as ready")
			Eventually(func() string {
				return helpers.CurlApp(Config, appName, "/ready")
			}, Config.DefaultTimeoutDuration()).Should(ContainSubstring("200 - ready"))

			// TODO: only include this when audit events are built
			// Eventually(func() string {
			// 	return string(cf.Cf("events", appName).Wait().Out.Contents())
			// }).Should(MatchRegexp("app.ready"))

			Expect(string(logs.Recent(appName).Wait().Out.Contents())).Should(ContainSubstring("Container passed the readiness health check"))

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

			By("using c2c to trigger the app to make the /ready endpoint succeed again")
			// push proxy app
			Expect(cf.Cf(
				"push", proxyAppName,
				"-b", Config.GetGoBuildpackName(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", assets.NewAssets().Proxy,
				"-f", assets.NewAssets().Proxy+"/manifest.yml",
			).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			// Make network policy for proxy to dora
			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				Expect(cf.Cf("target", "-o", orgName, "-s", spaceName).Wait()).To(Exit(0))
				Expect(cf.Cf("add-network-policy", proxyAppName, appName, "--protocol", "tcp", "--port", "8080").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
				Expect(string(cf.Cf("network-policies").Wait().Out.Contents())).To(ContainSubstring(appName))
			})

			// Wait until the c2c policy is in place
			Eventually(func() string {
				return helpers.CurlApp(Config, proxyAppName, fmt.Sprintf("/proxy/%s:8080", overlayIP))
			}, Config.DefaultTimeoutDuration()).Should(ContainSubstring("Dora"))

			// Trigger the app to make the /ready endpoint return a 200 again
			Eventually(func() string {
				return helpers.CurlApp(Config, proxyAppName, fmt.Sprintf("/proxy/%s:8080/ready/true", overlayIP))
			}).Should(ContainSubstring("true"))

			By("verifying that the routing table includes the app again")
			Eventually(func() string {
				return helpers.CurlApp(Config, appName, "/ready")
			}, readinessHealthCheckTimeout).Should(ContainSubstring("200 - ready"))

			By("verifying that the app hasn't restarted")
			Consistently(func() string {
				return string(cf.Cf("events", appName).Wait().Out.Contents())
			}).ShouldNot(MatchRegexp("audit.app.process.rescheduling"))

			// Verify that the overlay IP has not changed, and thus that the app has not been restaged
			Expect(helpers.CurlApp(Config, appName, "/myip")).To(Equal(overlayIP))
		})
	})
})
