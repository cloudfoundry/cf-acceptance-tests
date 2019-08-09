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

		Expect(cf.Push(appName,
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

	It("should be able to add and remove routes", func() {
		secondHost := generator.PrefixedRandomName(Config.GetNamePrefix(), "ROUTE")

		By("changing the environment")
		Eventually(cf.Cf("set-env", appName, "WHY", "force-app-update")).Should(Exit(0))

		By("adding a route")
		Eventually(cf.Cf("map-route", appName, Config.GetAppsDomain(), "-n", secondHost)).Should(Exit(0))
		Eventually(helpers.CurlingAppRoot(Config, appName)).Should(ContainSubstring("hello i am nora"))
		Eventually(helpers.CurlingAppRoot(Config, secondHost)).Should(ContainSubstring("hello i am nora"))

		By("removing a route")
		Eventually(cf.Cf("unmap-route", appName, Config.GetAppsDomain(), "-n", secondHost)).Should(Exit(0))
		Eventually(helpers.CurlingAppRoot(Config, secondHost)).Should(ContainSubstring("404"))
		Eventually(helpers.CurlingAppRoot(Config, appName)).Should(ContainSubstring("hello i am nora"))

		By("deleting the original route")
		Eventually(cf.Cf("delete-route", Config.GetAppsDomain(), "-n", appName, "-f")).Should(Exit(0))
		Eventually(helpers.CurlingAppRoot(Config, appName)).Should(ContainSubstring("404"))
	})
})
