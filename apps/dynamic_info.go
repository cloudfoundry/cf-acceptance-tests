package apps

import (
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
)

var _ = AppsDescribe("A running application", func() {
	var appName string

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")

		Expect(cf.Cf("push",
			appName,
			"-b", Config.GetBinaryBuildpackName(),
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", assets.NewAssets().Catnip,
			"-c", "./catnip",
			"-d", Config.GetAppsDomain()).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
	})

	AfterEach(func() {
		app_helpers.AppReport(appName, Config.DefaultTimeoutDuration())

		Expect(cf.Cf("delete", appName, "-f", "-r").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
	})

	It("shows crash events and recovers from crashes", func() {
		id := helpers.CurlApp(Config, appName, "/id")
		helpers.CurlApp(Config, appName, "/sigterm/KILL")

		Eventually(cf.Cf("events", appName)).Should(Say("[eE]xited"))

		Eventually(func() string {
			return helpers.CurlApp(Config, appName, "/id")
		}).Should(Not(Equal(id)))
	})

	Context("with multiple instances", func() {
		BeforeEach(func() {
			Expect(cf.Cf("scale", appName, "-i", "2").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		})

		It("can be queried for state by instance", func() {
			app := cf.Cf("app", appName).Wait(Config.DefaultTimeoutDuration())
			Expect(app).To(Exit(0))
			Expect(app).To(Say("#0"))
			Expect(app).To(Say("#1"))
		})
	})
})
