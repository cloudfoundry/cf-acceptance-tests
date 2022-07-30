package windows

import (
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers/skip_messages"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/helpers"
)

var _ = WindowsDescribe("Task Lifecycle", func() {
	var appName string

	BeforeEach(func() {
		if !Config.GetUseWindowsTestTask() {
			Skip(skip_messages.SkipWindowsTasksMessage)
		}

		appName = random_name.CATSRandomName("APP")

		Expect(cf.Cf("push",
			appName,
			"-s", Config.GetWindowsStack(),
			"-b", Config.GetBinaryBuildpackName(),
			"-c", ".\\webapp.exe",
			"-m", DEFAULT_WINDOWS_MEMORY_LIMIT,
			"-p", assets.NewAssets().WindowsWebapp,
		).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		Eventually(helpers.CurlingAppRoot(Config, appName)).Should(ContainSubstring("hi i am a standalone webapp"))
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)

		Expect(cf.Cf("delete", appName, "-f", "-r").Wait()).Should(Exit(0))
	})

	It("exercises the task lifecycle on windows", func() {
		session := cf.Cf("run-task", appName, "--command", "cmd /c echo 'hello world'")
		Eventually(session).Should(Exit(0))

		Eventually(func() *Session {
			taskSession := cf.Cf("tasks", appName)
			Expect(taskSession.Wait()).To(Exit(0))
			return taskSession
		}).Should(Say("SUCCEEDED"))
	})
})
