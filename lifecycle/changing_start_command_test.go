package lifecycle

import (
	"time"

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
		).To(SayWithTimeout("Started", 2*time.Minute))
	})

	AfterEach(func() {
		Expect(Cf("delete", AppName, "-f")).To(
			SayWithTimeout("OK", 30*time.Second),
		)
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

		restart := Cf("restart", AppName)

		Expect(restart).To(Say("Stopping"))
		Expect(restart).To(Say("OK"))

		Expect(restart).To(Say("Starting"))
		Expect(restart).To(Say("OK"))

		Expect(restart).To(Say("Started"))

		Eventually(Curling("/env/FOO")).Should(Say("bar"))
	})
})
