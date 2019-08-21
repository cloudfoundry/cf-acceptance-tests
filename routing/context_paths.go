package routing

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

var _ = RoutingDescribe("Context Paths", func() {
	var (
		app1              string
		helloRoutingAsset = assets.NewAssets().HelloRouting

		app2     string
		app2Path = "/app2"
		app3     string
		app3Path = "/app3/long/sub/path"
		hostname string
	)

	BeforeEach(func() {
		domain := Config.GetAppsDomain()

		app1 = random_name.CATSRandomName("APP")
		Expect(cf.Cf("push",
			app1,
			"-b", Config.GetRubyBuildpackName(),
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", helloRoutingAsset,
			"-d", Config.GetAppsDomain()).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		app2 = random_name.CATSRandomName("APP")
		Expect(cf.Cf("push",
			app2,
			"-b", Config.GetRubyBuildpackName(),
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", helloRoutingAsset,
			"-d", Config.GetAppsDomain()).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		app3 = random_name.CATSRandomName("APP")
		Expect(cf.Cf("push",
			app3,
			"-b", Config.GetRubyBuildpackName(),
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", helloRoutingAsset,
			"-d", Config.GetAppsDomain()).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

		hostname = app1

		Expect(cf.Cf("map-route", app2, domain, "--hostname", hostname, "--path", app2Path).Wait()).To(Exit(0))
		Expect(cf.Cf("map-route", app3, domain, "--hostname", hostname, "--path", app3Path).Wait()).To(Exit(0))
	})

	AfterEach(func() {
		app_helpers.AppReport(app1)
		app_helpers.AppReport(app2)
		app_helpers.AppReport(app3)
		Expect(cf.Cf("delete", app1, "-f", "-r").Wait()).To(Exit(0))
		Expect(cf.Cf("delete", app2, "-f", "-r").Wait()).To(Exit(0))
		Expect(cf.Cf("delete", app3, "-f", "-r").Wait()).To(Exit(0))
	})

	Context("when another app has a route with a context path", func() {
		It("routes to app with context path", func() {
			Eventually(func() string {
				return helpers.CurlAppRoot(Config, hostname)
			}).Should(ContainSubstring(app1))

			Eventually(func() string {
				return helpers.CurlApp(Config, hostname, app2Path)
			}).Should(ContainSubstring(app2))

			Eventually(func() string {
				return helpers.CurlApp(Config, hostname, app3Path)
			}).Should(ContainSubstring(app3))
		})
	})
})
