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
		domain := Config.AppsDomain

		app1 = random_name.CATSRandomName("APP")
		PushApp(app1, helloRoutingAsset, config.RubyBuildpackName, Config.AppsDomain, CF_PUSH_TIMEOUT)
		app2 = random_name.CATSRandomName("APP")
		PushApp(app2, helloRoutingAsset, config.RubyBuildpackName, Config.AppsDomain, CF_PUSH_TIMEOUT)
		app3 = random_name.CATSRandomName("APP")
		PushApp(app3, helloRoutingAsset, config.RubyBuildpackName, Config.AppsDomain, CF_PUSH_TIMEOUT)

		hostname = app1

		MapRouteToApp(app2, domain, hostname, app2Path, DEFAULT_TIMEOUT)
		MapRouteToApp(app3, domain, hostname, app3Path, DEFAULT_TIMEOUT)
	})

	AfterEach(func() {
		AppReport(app1, DEFAULT_TIMEOUT)
		AppReport(app2, DEFAULT_TIMEOUT)
		AppReport(app3, DEFAULT_TIMEOUT)

		DeleteApp(app1, DEFAULT_TIMEOUT)
		DeleteApp(app2, DEFAULT_TIMEOUT)
		DeleteApp(app3, DEFAULT_TIMEOUT)
	})

	Context("when another app has a route with a context path", func() {
		It("routes to app with context path", func() {
			Eventually(func() string {
				return helpers.CurlAppRoot(hostname)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring(app1))

			Eventually(func() string {
				return helpers.CurlApp(hostname, app2Path)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring(app2))

			Eventually(func() string {
				return helpers.CurlApp(hostname, app3Path)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring(app3))
		})
	})
})
