package windows

import (
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
)

var _ = WindowsDescribe("Setting an app's start command", func() {
	var appName string

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")

		Expect(cf.Cf("push",
			appName,
			"--no-route",
			"-s", Config.GetWindowsStack(),
			"-b", Config.GetBinaryBuildpackName(),
			"-m", DEFAULT_WINDOWS_MEMORY_LIMIT,
			"-c", "loop.bat Hi there!!!",
			"-u", "none",
			"-p", assets.NewAssets().BatchScript).Wait(Config.CfPushTimeoutDuration()),
		).To(Exit(0))
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)

		Expect(cf.Cf("delete", appName, "-f", "-r").Wait()).Should(Exit(0))
	})

	It("uses the given start command", func() {
		// OUT... to make sure we don't match the Launcher line: Running `loop.bat Hi there!!!'
		Eventually(func() *Buffer {
			return logs.Tail(Config.GetUseLogCache(), appName).Wait().Out
		}).Should(Say("OUT Hi there!!!"))
	})
})
