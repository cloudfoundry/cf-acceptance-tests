package apps

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	. "github.com/cloudfoundry/cf-acceptance-tests/helpers"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/generator"
)

var _ = Describe("Changing an app's start command", func() {
	var appName string

	BeforeEach(func() {
		appName = RandomName()

		Eventually(Cf(
			"push", appName,
			"-p", NewAssets().Dora,
			"-d", LoadConfig().AppsDomain,
			"-c", "FOO=foo bundle exec rackup config.ru -p $PORT",
		), CFPushTimeout).Should(Exit(0))
	})

	AfterEach(func() {
		Eventually(Cf("delete", appName, "-f"), DefaultTimeout).Should(Exit(0))
	})

	It("takes effect after a restart, not requiring a push", func() {
		Eventually(CurlFetcher(appName, "/env/FOO", LoadConfig().AppsDomain), DefaultTimeout).Should(ContainSubstring("foo"))

		var response QueryResponse

		ApiRequest("GET", "/v2/apps?q=name:"+appName, &response)

		Expect(response.Resources).To(HaveLen(1))

		appGuid := response.Resources[0].Metadata.Guid

		ApiRequest(
			"PUT",
			"/v2/apps/"+appGuid,
			nil,
			`{"command":"FOO=bar bundle exec rackup config.ru -p $PORT"}`,
		)

		Eventually(Cf("stop", appName), DefaultTimeout).Should(Exit(0))

		Eventually(CurlFetcher(appName, "/env/FOO", LoadConfig().AppsDomain), DefaultTimeout).Should(ContainSubstring("404"))

		Eventually(Cf("start", appName), DefaultTimeout).Should(Exit(0))

		Eventually(CurlFetcher(appName, "/env/FOO", LoadConfig().AppsDomain), DefaultTimeout).Should(ContainSubstring("bar"))
	})
})
