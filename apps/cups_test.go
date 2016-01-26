package apps

import (
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = FDescribe("User services", func() {
	var testConfig = helpers.LoadConfig()
	var appName string

	Describe("When a user-defined service is bound", func() {
		var serviceName string

		BeforeEach(func() {
			appName = generator.PrefixedRandomName("CATS-APP-")
			serviceName = generator.PrefixedRandomName("CUPS-")

			Eventually(cf.Cf(
				"push",
				appName,
				"--no-start",
				"-b", testConfig.RubyBuildpackName,
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", assets.NewAssets().RubySimple,
				"-d", testConfig.AppsDomain), DEFAULT_TIMEOUT).Should(Exit(0), "Failed to push app")

			app_helpers.SetBackend(appName)
			Expect(cf.Cf("start", appName).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

			Eventually(cf.Cf("cups", serviceName, "-l", "does this matter"), DEFAULT_TIMEOUT).Should(Exit(0), "Failed to create syslog drain service")
			Eventually(cf.Cf("bind-service", appName, serviceName), DEFAULT_TIMEOUT).Should(Exit(0), "Failed to bind service")
		})

		// No AfterEach because the delete occurs in the test itself

		It("can be deleted", func() {
			// AppReport here instead of AfterEach because the delete occurs in the test itself
			app_helpers.AppReport(appName, DEFAULT_TIMEOUT)

			Eventually(cf.Cf("delete", appName, "-f", "-r"), DEFAULT_TIMEOUT).Should(Exit(0), "Failed to delete app")
			if serviceName != "" {
				Eventually(cf.Cf("delete-service", serviceName, "-f"), DEFAULT_TIMEOUT).Should(Exit(0), "Failed to delete service")
			}
		})
	})
})
