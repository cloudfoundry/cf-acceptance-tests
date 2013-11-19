package apps

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/vito/cmdtest/matchers"

	. "github.com/pivotal-cf-experimental/cf-acceptance-tests/helpers"
)

var _ = Describe("An application being staged", func() {
	BeforeEach(func() {
		AppName = RandomName()
	})

	AfterEach(func() {
		Expect(Cf("delete", AppName, "-f")).To(Say("OK"))
	})

	It("has its staging log streamed during a push", func() {
		push := Cf("push", AppName, "-p", doraPath)

		Expect(push).To(Say("Staging..."))
		Expect(push).To(Say("Installing dependencies"))
		Expect(push).To(Say("Uploading droplet"))
		Expect(push).To(Say("Started"))
	})
})
