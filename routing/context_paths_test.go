package routing

import (
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Context Paths", func() {
	var (
		app1              string
		helloRoutingAsset = assets.NewAssets().HelloRouting

		app2     string
		app2Path = "/app2"
		app3     string
		app3Path = "/app3/long/sub/path"
		domain   string
	)

	BeforeEach(func() {
		app1 = PushApp(helloRoutingAsset, config.RubyBuildpackName)
		app2 = PushApp(helloRoutingAsset, config.RubyBuildpackName)
		app3 = PushApp(helloRoutingAsset, config.RubyBuildpackName)

		domain = app1

		MapRouteToApp(domain, app2Path, app2)
		MapRouteToApp(domain, app3Path, app3)
	})

	AfterEach(func() {
		app_helpers.AppReport(app1, DEFAULT_TIMEOUT)
		app_helpers.AppReport(app2, DEFAULT_TIMEOUT)
		app_helpers.AppReport(app3, DEFAULT_TIMEOUT)

		DeleteApp(app1)
		DeleteApp(app2)
		DeleteApp(app3)
	})

	Context("when another app has a route with a context path", func() {
		It("routes to app with context path", func() {
			Eventually(func() string {
				return helpers.CurlAppRoot(domain)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring(app1))

			Eventually(func() string {
				return helpers.CurlApp(domain, app2Path)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring(app2))

			Eventually(func() string {
				return helpers.CurlApp(domain, app3Path)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring(app3))
		})
	})
})
