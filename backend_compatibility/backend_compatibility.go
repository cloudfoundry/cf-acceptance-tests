package backend_compatibility

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

const binaryHi = "Hello from a binary"

var _ = BackendCompatibilityDescribe("Backend Compatibility", func() {
	var appName string

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")
		Eventually(cf.Cf(
			"push", appName,
			"-p", assets.NewAssets().Binary,
			"--no-start",
			"-m", DEFAULT_MEMORY_LIMIT,
			"-b", "binary_buildpack",
			"-d", Config.GetAppsDomain(),
			"-c", "./app"),
			Config.CfPushTimeoutDuration()).Should(Exit(0))
	})

	AfterEach(func() {
		app_helpers.AppReport(appName, Config.DefaultTimeoutDuration())
		Eventually(cf.Cf("delete", appName, "-f"), Config.DefaultTimeoutDuration()).Should(Exit(0))
	})

	Describe("An app staged on Diego", func() {
		BeforeEach(func() {
			app_helpers.EnableDiego(appName)

			Eventually(cf.Cf("start", appName), Config.CfPushTimeoutDuration()).Should(Exit(0))
			Eventually(helpers.CurlingAppRoot(Config, appName), Config.DefaultTimeoutDuration()).Should(ContainSubstring(binaryHi))
		})

		It("runs on the DEAs", func() {
			app_helpers.DisableDiego(appName)
			Eventually(cf.Cf("restart", appName), Config.CfPushTimeoutDuration()).Should(Exit(0))
			Eventually(helpers.CurlingAppRoot(Config, appName), Config.DefaultTimeoutDuration()).Should(ContainSubstring(binaryHi))
		})
	})

	Describe("An app staged on the DEA", func() {
		BeforeEach(func() {
			app_helpers.DisableDiego(appName)
			Eventually(cf.Cf("start", appName), Config.CfPushTimeoutDuration()).Should(Exit(0))
			Eventually(helpers.CurlingAppRoot(Config, appName), Config.DefaultTimeoutDuration()).Should(ContainSubstring(binaryHi))
		})

		It("runs on Diego", func() {
			app_helpers.EnableDiego(appName)
			Eventually(cf.Cf("restart", appName), Config.CfPushTimeoutDuration()).Should(Exit(0))
			Eventually(helpers.CurlingAppRoot(Config, appName), Config.DefaultTimeoutDuration()).Should(ContainSubstring(binaryHi))
		})
	})
})
