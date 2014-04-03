package diego

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/vito/cmdtest/matchers"

	. "github.com/cloudfoundry/cf-acceptance-tests/helpers"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/generator"
)

var _ = PDescribe("An application being staged with Diego", func() {
	var appName string

	BeforeEach(func() {
		appName = RandomName()
	})

	AfterEach(func() {
		Expect(Cf("delete", appName, "-f")).To(Say("OK"))
	})

	It("has its staging log streamed during a push", func() {
		Expect(Cf("push", appName, "-p", NewAssets().Dora, "--no-start")).To(ExitWith(0))
		Expect(Cf("set-env", appName, "CF_DIEGO_BETA", "true")).To(ExitWith(0))

		start := Cf("start", appName)

		Expect(start).To(Say("Downloading app package"))
		Expect(start).To(Say("Downloaded app package"))
		Expect(start).To(Say("Compiling"))
		Expect(start).To(Say("App started"))
	})
})
