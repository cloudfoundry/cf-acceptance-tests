package routing

import (
	. "github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/cloudfoundry-incubator/cf-routing-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	. "github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/onsi/ginkgo"
	. "github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/onsi/gomega"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
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

		app1 = GenerateAppName()
		PushApp(app1, helloRoutingAsset, config.RubyBuildpackName, config.AppsDomain, CF_PUSH_TIMEOUT)
		app2 = GenerateAppName()
		PushApp(app2, helloRoutingAsset, config.RubyBuildpackName, config.AppsDomain, CF_PUSH_TIMEOUT)
		app3 = GenerateAppName()
		PushApp(app3, helloRoutingAsset, config.RubyBuildpackName, config.AppsDomain, CF_PUSH_TIMEOUT)

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
