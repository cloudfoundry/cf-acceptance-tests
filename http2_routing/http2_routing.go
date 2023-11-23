package http2_routing

import (
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/v3_helpers"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = HTTP2RoutingDescribe("HTTP/2 apps", func() {
	var appName string

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")

		pushArgs := app_helpers.HTTP2WithArgs(appName, "--no-route", "-m", DEFAULT_MEMORY_LIMIT)
		Expect(cf.Cf(pushArgs...).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

		domain := Config.GetAppsDomain()
		Expect(cf.Cf("map-route", domain, "--hostname", appName, "--app-protocol", "http2").Wait()).To(Exit(0))
	})

	It("receive routed traffic over HTTP/2", func() {
		Eventually(helpers.CurlAppRoot).WithArguments(Config, appName).Should(ContainSubstring("Hello"))
	})
})
