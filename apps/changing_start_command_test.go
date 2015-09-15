package apps

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
)

var _ = Describe("Changing an app's start command", func() {
	var appName string

	BeforeEach(func() {
		appName = generator.PrefixedRandomName("CATS-APP-")

		Expect(cf.Cf(
			"push", appName,
			"-m", "128M",
			"-p", assets.NewAssets().Dora,
			"-d", helpers.LoadConfig().AppsDomain,
			"-c", "FOO=foo bundle exec rackup config.ru -p $PORT",
		).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
	})

	AfterEach(func() {
		Expect(cf.Cf("delete", appName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	})

	It("takes effect after a restart, not requiring a push", func() {
		Eventually(func() string {
			return helpers.CurlApp(appName, "/env/FOO")
		}, DEFAULT_TIMEOUT).Should(ContainSubstring("foo"))

		var response cf.QueryResponse

		cf.ApiRequest("GET", "/v2/apps?q=name:"+appName, &response, DEFAULT_TIMEOUT)

		Expect(response.Resources).To(HaveLen(1))

		appGuid := response.Resources[0].Metadata.Guid

		cf.ApiRequest(
			"PUT",
			"/v2/apps/"+appGuid,
			nil,
			DEFAULT_TIMEOUT,
			`{"command":"FOO=bar bundle exec rackup config.ru -p $PORT"}`,
		)

		Expect(cf.Cf("stop", appName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))

		Eventually(func() string {
			return helpers.CurlApp(appName, "/env/FOO")
		}, DEFAULT_TIMEOUT).Should(ContainSubstring("404"))

		Expect(cf.Cf("start", appName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))

		Eventually(func() string {
			return helpers.CurlApp(appName, "/env/FOO")
		}, DEFAULT_TIMEOUT).Should(ContainSubstring("bar"))
	})
})
