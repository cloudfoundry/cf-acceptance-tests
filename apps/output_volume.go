package apps

import (
	"time"

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

var _ = AppsDescribe("An application printing a bunch of output", func() {
	var appName string

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")

		Expect(cf.Cf("push", appName, "--no-start", "-b", Config.GetRubyBuildpackName(), "-m", DEFAULT_MEMORY_LIMIT, "-p", assets.NewAssets().Dora, "-d", Config.GetAppsDomain()).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
		app_helpers.SetBackend(appName)
		Expect(cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
	})

	AfterEach(func() {
		app_helpers.AppReport(appName, Config.DefaultTimeoutDuration())

		Expect(cf.Cf("delete", appName, "-f", "-r").Wait(Config.DefaultTimeoutDuration())).Should(Exit(0))
	})

	It("doesn't die when printing 32MB", func() {
		beforeId := helpers.CurlApp(Config, appName, "/id")

		Expect(helpers.CurlAppWithTimeout(Config, appName, "/logspew/32000", Config.LongCurlTimeoutDuration())).
			To(ContainSubstring("Just wrote 32000 kbytes to the log"))

		// Give time for components (i.e. Warden) to react to the output
		// and potentially make bad decisions (like killing the app)
		time.Sleep(10 * time.Second)

		afterId := helpers.CurlApp(Config, appName, "/id")

		Expect(beforeId).To(Equal(afterId))
	})
})
