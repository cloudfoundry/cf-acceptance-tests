package apps

import (
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
)

var _ = AppsDescribe("Delete Route", func() {
	var (
		appName              string
		expectedNullResponse string
	)

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")

		appUrl := "https://" + appName + "." + Config.GetAppsDomain()

		nullSession := helpers.CurlSkipSSL(Config.GetSkipSSLValidation(), appUrl).Wait(Config.DefaultTimeoutDuration())
		expectedNullResponse = string(nullSession.Buffer().Contents())

		Expect(cf.Cf("push", appName, "--no-start", "-b", Config.GetRubyBuildpackName(), "-m", DEFAULT_MEMORY_LIMIT, "-p", assets.NewAssets().Dora, "-d", Config.GetAppsDomain()).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
		app_helpers.SetBackend(appName)
		Expect(cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		Eventually(func() string {
			return helpers.CurlAppRoot(Config, appName)
		}, Config.DefaultTimeoutDuration()).Should(ContainSubstring("Hi, I'm Dora!"))
	})

	AfterEach(func() {
		app_helpers.AppReport(appName, Config.DefaultTimeoutDuration())

		Expect(cf.Cf("delete", appName, "-f", "-r").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
	})

	Describe("Removing the route", func() {
		It("Should be  able to remove and delete the route", func() {
			secondHost := random_name.CATSRandomName("ROUTE")

			By("adding a route")
			Eventually(cf.Cf("map-route", appName, Config.GetAppsDomain(), "-n", secondHost), Config.DefaultTimeoutDuration()).Should(Exit(0))
			Eventually(helpers.CurlingAppRoot(Config, appName), Config.DefaultTimeoutDuration()).Should(ContainSubstring("Hi, I'm Dora!"))
			Eventually(helpers.CurlingAppRoot(Config, secondHost), Config.DefaultTimeoutDuration()).Should(ContainSubstring("Hi, I'm Dora!"))

			By("removing a route")
			Eventually(cf.Cf("unmap-route", appName, Config.GetAppsDomain(), "-n", secondHost), Config.DefaultTimeoutDuration()).Should(Exit(0))
			Eventually(helpers.CurlingAppRoot(Config, secondHost), Config.DefaultTimeoutDuration()).ShouldNot(ContainSubstring("Hi, I'm Dora!"))
			Eventually(helpers.CurlingAppRoot(Config, appName), Config.DefaultTimeoutDuration()).Should(ContainSubstring("Hi, I'm Dora!"))

			By("deleting the original route")
			Expect(cf.Cf("delete-route", Config.GetAppsDomain(), "-n", appName, "-f").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
			Eventually(helpers.CurlingAppRoot(Config, appName), Config.DefaultTimeoutDuration()).Should(ContainSubstring(expectedNullResponse))
		})
	})
})
