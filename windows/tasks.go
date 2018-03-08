package windows

import (
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
)

var _ = WindowsDescribe("Task Lifecycle", func() {
	var appName string

	BeforeEach(func() {
		if !Config.GetUseWindowsTestTask() {
			Skip("Skipping tasks tests (requires diego-release v1.20.0 and above)")
		}

		appName = random_name.CATSRandomName("APP")

		Expect(cf.Cf("push",
			appName,
			"--no-start",
			"-s", Config.GetWindowsStack(),
			"-b", Config.GetBinaryBuildpackName(),
			"-c", ".\\webapp.exe",
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", assets.NewAssets().WindowsWebapp,
			"-d", Config.GetAppsDomain()).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
		app_helpers.SetBackend(appName)
		Expect(cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		Eventually(helpers.CurlingAppRoot(Config, appName)).Should(ContainSubstring("hi i am a standalone webapp"))
	})

	AfterEach(func() {
		app_helpers.AppReport(appName, Config.DefaultTimeoutDuration())

		Expect(cf.Cf("delete", appName, "-f", "-r").Wait(Config.DefaultTimeoutDuration())).Should(Exit(0))
	})

	It("exercises the task lifecycle on windows", func() {
		session := cf.Cf("run-task", appName, "cmd /c echo 'hello world'")
		Eventually(session).Should(Exit(0))

		Eventually(func() *Session {
			taskSession := cf.Cf("tasks", appName)
			Expect(taskSession.Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
			return taskSession
		}, Config.DefaultTimeoutDuration()).Should(Say("SUCCEEDED"))
	})
})
