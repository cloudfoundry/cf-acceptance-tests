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
	var appName string

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")

		Expect(cf.Cf("push", appName, "--no-start", "-b", Config.RubyBuildpackName, "-m", DEFAULT_MEMORY_LIMIT, "-p", assets.NewAssets().Dora, "-d", Config.AppsDomain).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		app_helpers.SetBackend(appName)
		Expect(cf.Cf("start", appName).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
		Eventually(func() string {
			return helpers.CurlAppRoot(appName)
		}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hi, I'm Dora!"))
	})

	AfterEach(func() {
		app_helpers.AppReport(appName, DEFAULT_TIMEOUT)

		Expect(cf.Cf("delete", appName, "-f", "-r").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	})

	Describe("Removing the route", func() {
		It("Should be  able to remove and delete the route", func() {
			secondHost := random_name.CATSRandomName("ROUTE")

			By("adding a route")
			Eventually(cf.Cf("map-route", appName, Config.AppsDomain, "-n", secondHost), DEFAULT_TIMEOUT).Should(Exit(0))
			Eventually(helpers.CurlingAppRoot(appName), DEFAULT_TIMEOUT).Should(ContainSubstring("Hi, I'm Dora!"))
			Eventually(helpers.CurlingAppRoot(secondHost), DEFAULT_TIMEOUT).Should(ContainSubstring("Hi, I'm Dora!"))

			By("removing a route")
			Eventually(cf.Cf("unmap-route", appName, Config.AppsDomain, "-n", secondHost), DEFAULT_TIMEOUT).Should(Exit(0))
			Eventually(helpers.CurlingAppRoot(secondHost), DEFAULT_TIMEOUT).Should(ContainSubstring("404"))
			Eventually(helpers.CurlingAppRoot(appName), DEFAULT_TIMEOUT).Should(ContainSubstring("Hi, I'm Dora!"))

			By("deleting the original route")
			Expect(cf.Cf("delete-route", Config.AppsDomain, "-n", appName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			Eventually(helpers.CurlingAppRoot(appName), DEFAULT_TIMEOUT).Should(ContainSubstring("404"))
		})
	})
})
