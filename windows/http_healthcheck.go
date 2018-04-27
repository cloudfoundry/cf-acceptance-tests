package windows

import (
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	logshelper "github.com/cloudfoundry/cf-acceptance-tests/helpers/logs"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = WindowsDescribe("Http Healthcheck", func() {
	var (
		appName string
		logs    *Session
	)

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")

		Expect(cf.Cf("push",
			appName,
			"--no-start",
			"-s", Config.GetWindowsStack(),
			"-b", Config.GetHwcBuildpackName(),
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", assets.NewAssets().Nora,
			"-d", Config.GetAppsDomain()).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
		app_helpers.SetBackend(appName)
		logs = logshelper.TailFollow(Config.GetUseLogCache(), appName)
	})

	AfterEach(func() {
		app_helpers.AppReport(appName, Config.DefaultTimeoutDuration())

		Expect(cf.Cf("delete", appName, "-f", "-r").Wait(Config.DefaultTimeoutDuration())).Should(Exit(0))
	})

	Describe("An app staged and running", func() {
		It("should not start with an invalid healthcheck endpoint", func() {
			Expect(cf.Cf("set-health-check", appName, "http", "--endpoint", "/invalidhealthcheck").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))

			start := cf.Cf("start", appName)
			defer start.Kill()
			Eventually(logs.Out, Config.CfPushTimeoutDuration()).Should(Say("health check never passed."))
		})

		It("starts with a valid http healthcheck endpoint", func() {
			Expect(cf.Cf("set-health-check", appName, "http", "--endpoint", "/healthcheck").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))

			Expect(cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			Eventually(helpers.CurlingAppRoot(Config, appName)).Should(ContainSubstring("hello i am nora"))
		})

		It("starts with a http healthcheck endpoint that is a redirect", func() {
			Expect(cf.Cf("set-health-check", appName, "http", "--endpoint", "/redirect/healthcheck").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))

			Expect(cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			Eventually(helpers.CurlingAppRoot(Config, appName)).Should(ContainSubstring("hello i am nora"))
		})

		It("does not start with a http healthcheck endpoint that is an invalid redirect", func() {
			Expect(cf.Cf("set-health-check", appName, "http", "--endpoint", "/redirect/invalidhealthcheck").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))

			start := cf.Cf("start", appName)
			defer start.Kill()
			Eventually(logs.Out, Config.CfPushTimeoutDuration()).Should(Say("health check never passed."))
		})
	})
})
