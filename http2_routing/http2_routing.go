package http2_routing

import (
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/v3_helpers"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = HTTP2RoutingDescribe("HTTP/2 Routing", func() {
	SkipOnK8s("Not yet supported in CF-for-K8s")

	Context("when a destination only supports HTTP/2", func() {
		It("routes traffic to that destination over HTTP/2", func() {
			appName := random_name.CATSRandomName("APP")

			Expect(cf.Cf(app_helpers.HTTP2WithArgs(
				appName,
				"--no-route",
				"-m", DEFAULT_MEMORY_LIMIT)...).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			appGUID := app_helpers.GetAppGuid(appName)

			Expect(cf.Cf("create-route", Config.GetAppsDomain(),
				"--hostname", appName,
			).Wait()).To(Exit(0))

			destination := Destination{
				App: App{
					GUID: appGUID,
				},
				HTTPVersion: 2,
			}
			InsertDestinations(GetRouteGuid(appName), []Destination{destination})

			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}).Should(ContainSubstring("Hello"))
		})
	})
})
