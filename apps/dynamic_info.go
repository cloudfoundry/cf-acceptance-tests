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
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/skip_messages"
)

var _ = AppsDescribe("A running application", func() {
	var appName string

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")

		Expect(cf.Cf("push",
			appName,
			"--no-start",
			"-b", Config.GetRubyBuildpackName(),
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", assets.NewAssets().Dora,
			"-d", Config.GetAppsDomain()).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		app_helpers.SetBackend(appName)
		Expect(cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
	})

	AfterEach(func() {
		app_helpers.AppReport(appName, Config.DefaultTimeoutDuration())

		Expect(cf.Cf("delete", appName, "-f", "-r").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
	})

	It("can have its files inspected", func() {
		// Currently cannot work with multiple instances since CF always checks instance 0
		if Config.GetBackend() != "dea" {
			Skip(skip_messages.SkipDeaMessage)
		}
		files := cf.Cf("files", appName).Wait(Config.DefaultTimeoutDuration())
		Expect(files).To(Exit(0))
		Expect(files).To(Say("app/"))

		files = cf.Cf("files", appName, "app/").Wait(Config.DefaultTimeoutDuration())
		Expect(files).To(Exit(0))
		Expect(files).To(Say("config.ru"))

		files = cf.Cf("files", appName, "app/config.ru").Wait(Config.DefaultTimeoutDuration())
		Expect(files).To(Exit(0))
		Expect(files).To(Say("run Dora"))
	})

	It("shows crash events and recovers from crashes", func() {
		id := helpers.CurlApp(Config, appName, "/id")
		helpers.CurlApp(Config, appName, "/sigterm/KILL")

		Eventually(func() string {
			return string(cf.Cf("events", appName).Wait(Config.DefaultTimeoutDuration()).Out.Contents())
		}, Config.DefaultTimeoutDuration()).Should(MatchRegexp("[eE]xited"))

		Eventually(func() string { return helpers.CurlApp(Config, appName, "/id") }).Should(Not(Equal(id)))
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
