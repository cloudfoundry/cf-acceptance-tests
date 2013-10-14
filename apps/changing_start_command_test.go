package apps

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/vito/cmdtest/matchers"

	. "github.com/vito/runtime-integration/helpers"
)

var _ = Describe("Changing an app's start command", func() {
	BeforeEach(func() {
		AppName = RandomName()

		Expect(
			Cf(
				"push", AppName,
				"-p", doraPath,
				"-c", "FOO=foo bundle exec rackup config.ru -p $PORT",
			),
		).To(Say("Started"))
	})

	AfterEach(func() {
		Expect(Cf("delete", AppName, "-f")).To(Say("OK"))
	})

	It("takes effect after a restart, not requiring a push", func() {
		Eventually(Curling("/env/FOO")).Should(Say("foo"))

		var response AppQueryResponse

		ApiRequest("GET", "/v2/apps?q=name:" + AppName, &response)

		Expect(response.Resources).To(HaveLen(1))

		appGuid := response.Resources[0].Metadata.Guid

		ApiRequest(
			"PUT",
			"/v2/apps/" + appGuid,
			nil,
			`{"command":"FOO=bar bundle exec rackup config.ru -p $PORT"}`,
		)

		Expect(Cf("stop", AppName)).To(Say("OK"))

		Eventually(Curling("/env/FOO")).Should(Say("404"))

		Expect(Cf("start", AppName)).To(Say("Started"))

		Eventually(Curling("/env/FOO")).Should(Say("bar"))
	})
})
