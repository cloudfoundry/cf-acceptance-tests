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

var _ = AppsDescribe("Crashing", func() {
	var appName string

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")
	})

	AfterEach(func() {
		app_helpers.AppReport(appName, Config.DefaultTimeoutDuration())
		Expect(cf.Cf("delete", appName, "-f", "-r").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
	})

	Describe("a continuously crashing app", func() {
		BeforeEach(func() {
			if Config.GetBackend() != "diego" {
				Skip(skip_messages.SkipDiegoMessage)
			}
		})

		It("emits crash events and reports as 'crashed' after enough crashes", func() {
			Expect(cf.Cf(
				"push",
				appName,
				"-c", "/bin/false",
				"--no-start",
				"-b", Config.GetRubyBuildpackName(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", assets.NewAssets().Dora,
				"-d", Config.GetAppsDomain(),
			).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			app_helpers.SetBackend(appName)
			Expect(cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(1))

			Eventually(func() string {
				return string(cf.Cf("events", appName).Wait(Config.DefaultTimeoutDuration()).Out.Contents())
			}, Config.DefaultTimeoutDuration()).Should(MatchRegexp("[eE]xited"))

			Eventually(cf.Cf("app", appName), Config.DefaultTimeoutDuration()).Should(Say("crashed"))
		})
	})

	Context("the app crashes", func() {
		BeforeEach(func() {
			Expect(cf.Cf(
				"push",
				appName,
				"--no-start",
				"-b", Config.GetRubyBuildpackName(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", assets.NewAssets().Dora,
				"-d", Config.GetAppsDomain(),
			).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))

			app_helpers.SetBackend(appName)
			Expect(cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		})

		It("shows crash events", func() {
			helpers.CurlApp(Config, appName, "/sigterm/KILL")

			Eventually(func() string {
				return string(cf.Cf("events", appName).Wait(Config.DefaultTimeoutDuration()).Out.Contents())
			}, Config.DefaultTimeoutDuration()).Should(MatchRegexp("[eE]xited"))
		})

		It("recovers", func() {
			id := helpers.CurlApp(Config, appName, "/id")
			helpers.CurlApp(Config, appName, "/sigterm/KILL")

			Eventually(func() string {
				return helpers.CurlApp(Config, appName, "/id")
			}, Config.DefaultTimeoutDuration()).Should(Not(Equal(id)))
		})
	})
})
