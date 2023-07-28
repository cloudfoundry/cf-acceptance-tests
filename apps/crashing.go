package apps

import (
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/helpers"
)

var _ = AppsDescribe("Crashing", func() {
	var appName string

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)
		Expect(cf.Cf("delete", appName, "-f", "-r").Wait()).To(Exit(0))
	})

	Describe("a continuously crashing app", func() {
		It("emits crash events and reports as 'crashed' after enough crashes", func() {
			Expect(cf.Cf(
				"push",
				appName,
				"-c", "/bin/false",
				"-b", Config.GetBinaryBuildpackName(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", assets.NewAssets().Catnip,
			).Wait(Config.CfPushTimeoutDuration())).To(Exit(1))

			Eventually(func() string {
				return string(cf.Cf("events", appName).Wait().Out.Contents())
			}).Should(MatchRegexp("app.crash"))

			Eventually(cf.Cf("app", appName)).Should(Say("crashed"))
		})
	})

	Context("the app crashes", func() {
		BeforeEach(func() {
			Expect(cf.Cf(app_helpers.CatnipWithArgs(
				appName,
				"-m", DEFAULT_MEMORY_LIMIT)...,
			).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		})

		It("shows crash events", func() {
			helpers.CurlApp(Config, appName, "/sigterm/KILL")
			Eventually(func() string {
				return string(cf.Cf("events", appName).Wait().Out.Contents())
			}).Should(MatchRegexp("app.crash"))
		})

		It("recovers", func() {
			const idChecker = "^[0-9a-zA-Z]+(?:-[0-9a-zA-z]+)+$"

			id := helpers.CurlApp(Config, appName, "/id")
			Expect(id).Should(MatchRegexp(idChecker))
			helpers.CurlApp(Config, appName, "/sigterm/KILL")

			Eventually(func() string {
				return helpers.CurlApp(Config, appName, "/id")
			}).Should(MatchRegexp(idChecker))
		})
	})
})
