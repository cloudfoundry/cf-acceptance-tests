package routing

import (
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/v3_helpers"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = RoutingDescribe("Process Routing", func() {
	var (
		appName              string
		exisitingProcessGuid string
		canaryProcessGuid    string
		consistentlyDuration = "1s"
	)

	BeforeEach(func() {
		By("Pushing an application")
		appName = random_name.CATSRandomName("APP")
		Expect(cf.Cf("push", appName, "-i", "2", "-b", Config.GetRubyBuildpackName(), "-p", assets.NewAssets().DoraZip).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

		By("Waiting until all instances are running")
		Eventually(func(g Gomega) {
			session := cf.Cf("app", appName).Wait()
			g.Expect(session).Should(Say(`instances:\s+3/3`))
		})
		Eventually(func() string {
			return helpers.CurlAppRoot(Config, appName)
		}).Should(ContainSubstring("Hi, I'm Dora"))

		By("Pushing a canary deployment")
		Expect(cf.Cf("push", appName, "--strategy", "canary", "--no-wait", "-b", Config.GetStaticFileBuildpackName(), "-p", assets.NewAssets().Staticfile).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

		By("Waiting until canary is paused")
		Eventually(func(g Gomega) {
			session := cf.Cf("app", appName).Wait()
			g.Expect(session).Should(Say("Active deployment with status PAUSED"))
		}).Should(Succeed())

		appGuid := GetApp(appName).GUID
		processGuids := GetProcessGuidsForType(appGuid, "web")
		Expect(processGuids).To(HaveLen(2))
		exisitingProcessGuid = processGuids[0]
		canaryProcessGuid = processGuids[1]
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)
		Expect(cf.Cf("delete", appName, "-f").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
	})

	Describe("the X-CF-PROCESS-INSTANCE header", func() {
		It("can be used to route requests to the correct process and instance", func() {

			By("verifying that without the header, requests are routed randomly to either process")
			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}).Should(ContainSubstring("Hello from a staticfile"))

			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}).Should(ContainSubstring("Hi, I'm Dora"))

			By("verifying that with X-CF-PROCESS-INSTANCE set without an instance id, requests are routed to the correct process")
			existingProcessRoutingHeader := []string{"-H", "X-CF-PROCESS-INSTANCE:" + exisitingProcessGuid}
			canaryProcessRoutingHeader := []string{"-H", "X-CF-PROCESS-INSTANCE:" + canaryProcessGuid}

			By("checking that we can exclusively route to the existing process")
			Consistently(func() string {
				return helpers.CurlApp(Config, appName, "/", existingProcessRoutingHeader...)
			}, consistentlyDuration).Should(ContainSubstring("Hi, I'm Dora"))

			Consistently(func() string {
				return helpers.CurlApp(Config, appName, "/", existingProcessRoutingHeader...)
			}, consistentlyDuration).ShouldNot(ContainSubstring("Hello from a staticfile"))

			By("checking that we can exclusively route to the canary process")
			Consistently(func() string {
				return helpers.CurlApp(Config, appName, "/", canaryProcessRoutingHeader...)
			}, consistentlyDuration).Should(ContainSubstring("Hello from a staticfile"))

			Consistently(func() string {
				return helpers.CurlApp(Config, appName, "/", canaryProcessRoutingHeader...)
			}, consistentlyDuration).ShouldNot(ContainSubstring("Hi, I'm Dora"))

			By("verifying that with X-CF-PROCESS-INSTANCE set with an instance id, requests are routed to the correct process instance")
			routingHeaderFirstInstance := []string{"-H", "X-CF-PROCESS-INSTANCE:" + exisitingProcessGuid + ":0"}
			routingHeaderSecondInstance := []string{"-H", "X-CF-PROCESS-INSTANCE:" + exisitingProcessGuid + ":1"}
			firstInstanceId := helpers.CurlApp(Config, appName, "/id", routingHeaderFirstInstance...)

			By("checking that we can exclusively route to the first existing process")
			Consistently(func() string {
				return helpers.CurlApp(Config, appName, "/", routingHeaderFirstInstance...)
			}, consistentlyDuration).Should(ContainSubstring("Hi, I'm Dora"))

			Consistently(func() string {
				return helpers.CurlApp(Config, appName, "/id", routingHeaderFirstInstance...)
			}, consistentlyDuration).Should(Equal(firstInstanceId))

			secondInstanceId := helpers.CurlApp(Config, appName, "/id", routingHeaderSecondInstance...)

			By("checking that we can exclusively route to the second existing process")
			Consistently(func() string {
				return helpers.CurlApp(Config, appName, "/", routingHeaderSecondInstance...)
			}, consistentlyDuration).Should(ContainSubstring("Hi, I'm Dora"))

			Consistently(func() string {
				return helpers.CurlApp(Config, appName, "/id", routingHeaderSecondInstance...)
			}, consistentlyDuration).Should(Equal(secondInstanceId))
		})
	})
})
