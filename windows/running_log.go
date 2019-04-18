package windows

import (
	"fmt"
	"net/url"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/logs"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
)

var _ = WindowsDescribe("app logs", func() {
	var appName string

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")

		Expect(cf.Cf("push",
			appName,
			"-s", Config.GetWindowsStack(),
			"-b", Config.GetHwcBuildpackName(),
			"-m", DEFAULT_WINDOWS_MEMORY_LIMIT,
			"-p", assets.NewAssets().Nora,
			"-d", Config.GetAppsDomain()).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		Eventually(helpers.CurlingAppRoot(Config, appName)).Should(ContainSubstring("hello i am nora"))
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)

		Expect(cf.Cf("delete", appName, "-f", "-r").Wait()).Should(Exit(0))
	})

	It("captures stdout logs with the correct tag", func() {
		var message string

		By("logging application stdout")
		message = "message-from-stdout"
		helpers.CurlApp(Config, appName, fmt.Sprintf("/print/%s", url.QueryEscape(message)))

		Eventually(func() *Buffer {
			return logs.Tail(Config.GetUseLogCache(), appName).Wait().Out
		}).Should(Say(fmt.Sprintf("\\[APP(.*)/0\\]\\s*OUT %s", message)))
	})

	It("captures stderr logs with the correct tag", func() {
		var message string

		By("logging application stderr")
		message = "message-from-stderr"
		helpers.CurlApp(Config, appName, fmt.Sprintf("/print_err/%s", url.QueryEscape(message)))

		Eventually(func() *Buffer {
			return logs.Tail(Config.GetUseLogCache(), appName).Wait().Out
		}).Should(Say(fmt.Sprintf("\\[APP(.*)/0\\]\\s*ERR %s", message)))
	})
})
