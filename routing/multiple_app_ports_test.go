package routing

import (
	"fmt"

	. "github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/cloudfoundry-incubator/cf-routing-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	. "github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/onsi/ginkgo"
	. "github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/onsi/gomega"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
)

var _ = Describe(deaUnsupportedTag+"Multiple App Ports", func() {
	var (
		app             string
		secondRoute     string
		latticeAppAsset = assets.NewAssets().LatticeApp
	)

	BeforeEach(func() {
		app = GenerateAppName()
		cmd := fmt.Sprintf("lattice-app --ports=7777,8888,8080")

		PushAppNoStart(app, latticeAppAsset, config.GoBuildpackName, config.AppsDomain, CF_PUSH_TIMEOUT, "-c", cmd)
		EnableDiego(app, DEFAULT_TIMEOUT)
		StartApp(app, APP_START_TIMEOUT)
	})

	AfterEach(func() {
		AppReport(app, DEFAULT_TIMEOUT)
		DeleteApp(app, DEFAULT_TIMEOUT)
	})

	Context("when app only has single route", func() {
		Context("when no ports are specified for the app", func() {
			It("should listen on the default app port", func() {
				Eventually(func() string {
					return helpers.CurlApp(app, "/port")
				}, DEFAULT_TIMEOUT, "5s").Should(ContainSubstring("8080"))
			})
		})
	})

	Context("when app has multiple ports mapped", func() {
		BeforeEach(func() {
			UpdatePorts(app, []uint16{7777, 8888, 8080}, DEFAULT_TIMEOUT)
			// create 2nd route
			spacename := context.RegularUserContext().Space
			secondRoute = fmt.Sprintf("%s-two", app)
			CreateRoute(secondRoute, "", spacename, config.AppsDomain, DEFAULT_TIMEOUT)

			// map app route to other port
			CreateRouteMapping(app, secondRoute, 0, 7777, DEFAULT_TIMEOUT)
		})

		It("should listen on multiple ports", func() {
			Eventually(func() string {
				return helpers.CurlApp(app, "/port")
			}, DEFAULT_TIMEOUT, "5s").ShouldNot(Equal(""))

			Consistently(func() string {
				return helpers.CurlApp(app, "/port")
			}, DEFAULT_TIMEOUT, "5s").Should(ContainSubstring("8080"))

			Eventually(func() string {
				return helpers.CurlApp(secondRoute, "/port")
			}, DEFAULT_TIMEOUT, "5s").Should(ContainSubstring("7777"))
		})
	})
})
