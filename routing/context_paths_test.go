package routing

import (
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
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
		hostname string
	)

	BeforeEach(func() {
		domain := config.AppsDomain

		app1 = PushApp(helloRoutingAsset, config.RubyBuildpackName)
		app2 = PushApp(helloRoutingAsset, config.RubyBuildpackName)
		app3 = PushApp(helloRoutingAsset, config.RubyBuildpackName)

		hostname = app1

		MapRouteToApp(app2, domain, hostname, app2Path)
		MapRouteToApp(app3, domain, hostname, app3Path)
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
