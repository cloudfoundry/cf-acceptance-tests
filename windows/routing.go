package windows

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
)

var _ = WindowsDescribe("Adding and removing routes", func() {
	var appName string

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
		Expect(cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		Eventually(helpers.CurlingAppRoot(Config, appName)).Should(ContainSubstring("hello i am nora"))
	})

	AfterEach(func() {
		app_helpers.AppReport(appName, Config.DefaultTimeoutDuration())

		Expect(cf.Cf("delete", appName, "-f", "-r").Wait(Config.DefaultTimeoutDuration())).Should(Exit(0))
	})

	It("should be able to add and remove routes", func() {
		secondHost := generator.PrefixedRandomName(Config.GetNamePrefix(), "ROUTE")

		By("changing the environment")
		Eventually(cf.Cf("set-env", appName, "WHY", "force-app-update")).Should(Exit(0))

		By("adding a route")
		Eventually(cf.Cf("map-route", appName, Config.GetAppsDomain(), "-n", secondHost), Config.DefaultTimeoutDuration()).Should(Exit(0))
		Eventually(helpers.CurlingAppRoot(Config, appName)).Should(ContainSubstring("hello i am nora"))
		Eventually(helpers.CurlingAppRoot(Config, secondHost)).Should(ContainSubstring("hello i am nora"))

		By("removing a route")
		Eventually(cf.Cf("unmap-route", appName, Config.GetAppsDomain(), "-n", secondHost), Config.DefaultTimeoutDuration()).Should(Exit(0))
		Eventually(helpers.CurlingAppRoot(Config, secondHost)).Should(ContainSubstring("404"))
		Eventually(helpers.CurlingAppRoot(Config, appName)).Should(ContainSubstring("hello i am nora"))

		By("deleting the original route")
		Eventually(cf.Cf("delete-route", Config.GetAppsDomain(), "-n", appName, "-f"), Config.DefaultTimeoutDuration()).Should(Exit(0))
		Eventually(helpers.CurlingAppRoot(Config, appName)).Should(ContainSubstring("404"))
	})
})
