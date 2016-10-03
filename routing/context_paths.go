package routing

import (
	. "code.cloudfoundry.org/cf-routing-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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
		PushApp(app1, helloRoutingAsset, Config.GetRubyBuildpackName(), Config.GetAppsDomain(), Config.CfPushTimeoutDuration(), DEFAULT_MEMORY_LIMIT)
		app2 = random_name.CATSRandomName("APP")
		PushApp(app2, helloRoutingAsset, Config.GetRubyBuildpackName(), Config.GetAppsDomain(), Config.CfPushTimeoutDuration(), DEFAULT_MEMORY_LIMIT)
		app3 = random_name.CATSRandomName("APP")
		PushApp(app3, helloRoutingAsset, Config.GetRubyBuildpackName(), Config.GetAppsDomain(), Config.CfPushTimeoutDuration(), DEFAULT_MEMORY_LIMIT)

		hostname = app1

		MapRouteToApp(app2, domain, hostname, app2Path, Config.DefaultTimeoutDuration())
		MapRouteToApp(app3, domain, hostname, app3Path, Config.DefaultTimeoutDuration())
	})

	AfterEach(func() {
		AppReport(app1, Config.DefaultTimeoutDuration())
		AppReport(app2, Config.DefaultTimeoutDuration())
		AppReport(app3, Config.DefaultTimeoutDuration())

		DeleteApp(app1, Config.DefaultTimeoutDuration())
		DeleteApp(app2, Config.DefaultTimeoutDuration())
		DeleteApp(app3, Config.DefaultTimeoutDuration())
	})

	Context("when another app has a route with a context path", func() {
		It("routes to app with context path", func() {
			Eventually(func() string {
				return helpers.CurlAppRoot(Config, hostname)
			}, Config.DefaultTimeoutDuration()).Should(ContainSubstring(app1))

			Eventually(func() string {
				return helpers.CurlApp(Config, hostname, app2Path)
			}, Config.DefaultTimeoutDuration()).Should(ContainSubstring(app2))

			Eventually(func() string {
				return helpers.CurlApp(Config, hostname, app3Path)
			}, Config.DefaultTimeoutDuration()).Should(ContainSubstring(app3))
		})
	})
})
