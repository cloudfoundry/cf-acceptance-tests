package routing

import (
	"fmt"

	"github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	. "github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/onsi/ginkgo"
	. "github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/onsi/gomega"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/routing_helpers"
)

var _ = Describe(deaUnsupportedTag+"Multiple App Ports", func() {
	var (
		app             string
		secondRoute     string
		latticeAppAsset = assets.NewAssets().LatticeApp
	)

	BeforeEach(func() {
		app = GenerateAppName()
		cmd := fmt.Sprintf("%s --ports=7777,8080", "lattice-app")

		// will creates a route for default port 8080
		PushAppNoStart(app, latticeAppAsset, config.GoBuildpackName, config.AppsDomain, CF_PUSH_TIMEOUT, "-c", cmd)
		app_helpers.EnableDiego(app)
		UpdatePorts(app, []uint32{7777, 8080}, DEFAULT_TIMEOUT)
		StartApp(app, DEFAULT_TIMEOUT)
	})

	AfterEach(func() {
		app_helpers.AppReport(app, DEFAULT_TIMEOUT)
		DeleteApp(app, DEFAULT_TIMEOUT)
	})

	Context("when app only has single route", func() {
		It("should listen on the default app port", func() {
			Eventually(func() string {
				return helpers.CurlApp(app, "/port")
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("8080"))
		})
	})

	Context("when app has multiple ports mapped", func() {
		BeforeEach(func() {
			// create 2nd route
			domain := config.AppsDomain
			spacename := context.RegularUserContext().Space
			secondRoute = fmt.Sprintf("%s-two", app)
			CreateRoute(secondRoute, "", spacename, domain, DEFAULT_TIMEOUT)

			// map app route to other port
			CreateRouteMapping(app, secondRoute, 7777, DEFAULT_TIMEOUT)
		})

		It("should listen on multiple ports", func() {
			Eventually(func() string {
				return helpers.CurlApp(app, "/port")
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("8080"))

			Eventually(func() string {
				return helpers.CurlApp(secondRoute, "/port")
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("7777"))
		})

		It("returns an error when switching from Diego to DEA", func() {
			app_helpers.DisableDiegoAndCheckResponse(app, "CF-MultipleAppPortsMappedDiegoToDea")
		})
	})
})
