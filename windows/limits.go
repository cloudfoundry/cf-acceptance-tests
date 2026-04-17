package windows

import (
	"regexp"
	"strings"
	"time"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	logshelper "github.com/cloudfoundry/cf-acceptance-tests/helpers/logs"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = WindowsDescribe("App Limits", func() {
	var appName string

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")

		Expect(cf.Cf("push",
			appName,
			"-s", Config.GetWindowsStack(),
			"-b", Config.GetHwcBuildpackName(),
			"-m", "512m",
			"-p", assets.NewAssets().Nora,
			"-l", "1K",
		).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		Eventually(helpers.CurlingAppRoot(Config, appName)).Should(ContainSubstring("hello i am nora"))
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)

		Expect(cf.Cf("delete", appName, "-f", "-r").Wait()).Should(Exit(0))
	})

	It("does not allow the app to use more memory than allowed", func() {
		response := helpers.CurlApp(Config, appName, "/leakmemory/600")
		Expect(response).To(ContainSubstring("Insufficient memory"))
	})

	Context("when a log rate limit is defined", func() {
		var logs *Session

		BeforeEach(func() {
			logs = logshelper.Follow(appName)
		})

		AfterEach(func() {
			// logs might be nil if the BeforeEach panics
			if logs != nil {
				logs.Interrupt()
			}
		})

		It("enforces the log rate limit", func() {
			helpers.CurlApp(Config, appName, "/logspew/2")
			Eventually(logs).Should(Say(strings.Repeat("1", 1024)))
			Eventually(logs).Should(Say(regexp.QuoteMeta("app instance exceeded log rate limit (1024 bytes/sec)")), "log rate limit not enforced")
			Consistently(logs).ShouldNot(Say("11111"), "logs above the limit were not dropped")

			By("sleeping so that the app is allowed to output more logs")
			time.Sleep(time.Second)

			helpers.CurlApp(Config, appName, "/logspew/2")
			Eventually(logs).Should(Say(strings.Repeat("1", 1024)))
			Eventually(logs).Should(Say(regexp.QuoteMeta("app instance exceeded log rate limit (1024 bytes/sec)")), "log rate limit not enforced")
			Consistently(logs).ShouldNot(Say("11111"), "logs above the limit were not dropped")
		})
	})
})
