package windows

import (
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = WindowsDescribe("Http Healthcheck", func() {
	var (
		appName string
	)

	AfterEach(func() {
		app_helpers.AppReport(appName)

		Expect(cf.Cf("delete", appName, "-f", "-r").Wait()).Should(Exit(0))
	})

	Describe("An app staged and running", func() {
		Context("when the healthcheck endpoint normal", func() {
			BeforeEach(func() {
				appName = random_name.CATSRandomName("APP")

				Expect(cf.Cf("push",
					appName,
					"-s", Config.GetWindowsStack(),
					"-b", Config.GetHwcBuildpackName(),
					"-m", DEFAULT_WINDOWS_MEMORY_LIMIT,
					"-p", assets.NewAssets().Nora,
					"--health-check-type", "http",
					"--endpoint", "/healthcheck",
				).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
			})

			It("starts with a valid http healthcheck endpoint", func() {
				Eventually(helpers.CurlingAppRoot(Config, appName)).Should(ContainSubstring("hello i am nora"))
			})
		})

		Context("when the endpoint is a redirect", func() {
			BeforeEach(func() {
				appName = random_name.CATSRandomName("APP")

				Expect(cf.Cf("push",
					appName,
					"-s", Config.GetWindowsStack(),
					"-b", Config.GetHwcBuildpackName(),
					"-m", DEFAULT_WINDOWS_MEMORY_LIMIT,
					"-p", assets.NewAssets().Nora,
					"--health-check-type", "http",
					"--endpoint", "/redirect/healthcheck",
				).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
			})

			It("starts with a http healthcheck endpoint that is a redirect", func() {
				Eventually(helpers.CurlingAppRoot(Config, appName)).Should(ContainSubstring("hello i am nora"))
			})
		})
	})
})
