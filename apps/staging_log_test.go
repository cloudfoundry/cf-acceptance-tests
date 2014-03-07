package apps

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/vito/cmdtest/matchers"

	. "github.com/cloudfoundry/cf-acceptance-tests/helpers"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/generator"
)

var _ = Describe("An application being staged", func() {
	var appName string

	BeforeEach(func() {
		appName = RandomName()
	})

	AfterEach(func() {
		Expect(Cf("delete", appName, "-f")).To(Say("OK"))
	})

	It("has its staging log streamed during a push", func() {
		push := Cf("push", appName, "-p", NewAssets().Dora)

		// Expect(push).To(Say("Installing dependencies"))
		Expect(push).To(Say("Uploading droplet"))
		Expect(push).To(Say("App started"))
	})
})
