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

		nullSession := helpers.CurlSkipSSL(Config.GetSkipSSLValidation(), appUrl).Wait()
		expectedNullResponse = string(nullSession.Buffer().Contents())

		Expect(cf.Push(appName,
			"-b", Config.GetBinaryBuildpackName(),
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", assets.NewAssets().Catnip,
			"-c", "./catnip",
			"-d", Config.GetAppsDomain()).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		Eventually(func() string {
			return helpers.CurlAppRoot(Config, appName)
		}).Should(ContainSubstring("Catnip?"))
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)

		Expect(cf.Cf("delete", appName, "-f", "-r").Wait()).To(Exit(0))
	})

	Describe("Removing the route", func() {
		It("Should be  able to remove and delete the route", func() {
			secondHost := random_name.CATSRandomName("ROUTE")

			By("adding a route")
			Eventually(cf.Cf("map-route", appName, Config.GetAppsDomain(), "-n", secondHost)).Should(Exit(0))
			Eventually(helpers.CurlingAppRoot(Config, appName)).Should(ContainSubstring("Catnip?"))
			Eventually(helpers.CurlingAppRoot(Config, secondHost)).Should(ContainSubstring("Catnip?"))

			By("removing a route")
			Eventually(cf.Cf("unmap-route", appName, Config.GetAppsDomain(), "-n", secondHost)).Should(Exit(0))
			Eventually(helpers.CurlingAppRoot(Config, secondHost)).ShouldNot(ContainSubstring("Catnip?"))
			Eventually(helpers.CurlingAppRoot(Config, appName)).Should(ContainSubstring("Catnip?"))

			By("deleting the original route")
			Expect(cf.Cf("delete-route", Config.GetAppsDomain(), "-n", appName, "-f").Wait()).To(Exit(0))
			Eventually(helpers.CurlingAppRoot(Config, appName)).Should(ContainSubstring(expectedNullResponse))
		})
	})
})
