package apps

import (
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("FUSE", func() {

	config := helpers.LoadConfig()
	if config.IncludePrivilegedContainerSupport {

		var appName string

		BeforeEach(func() {
			appName = random_name.CATSRandomName("APP")
		})

		AfterEach(func() {
			app_helpers.AppReport(appName, DEFAULT_TIMEOUT)

			Expect(cf.Cf("delete", appName, "-f", "-r").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		})

		It("Can mount a fuse endpoint", func() {
			Expect(cf.Cf("push", appName, "--no-start", "-b", config.RubyBuildpackName, "-m", DEFAULT_MEMORY_LIMIT, "-p", assets.NewAssets().Fuse, "-d", config.AppsDomain).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			app_helpers.SetBackend(appName)
			Expect(cf.Cf("start", appName).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("great success!"))
		})
	}
})
