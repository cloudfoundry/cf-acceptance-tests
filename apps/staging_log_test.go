package apps

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vito/cmdtest"
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

		stagingLogsShown := false
		Expect(push).To(SayBranches(
			cmdtest.ExpectBranch{
				Pattern: "Downloaded app package",
				Callback: func() {
					stagingLogsShown = true
				},
			},
			cmdtest.ExpectBranch{
				Pattern: "Installing dependencies",
				Callback: func() {
					stagingLogsShown = true
				},
			},
			cmdtest.ExpectBranch{
				Pattern: "Uploading droplet",
				Callback: func() {
					stagingLogsShown = true
				},
			},
		))

		Expect(stagingLogsShown).To(BeTrue())
		Expect(push).To(Say("App started"))
	})
})
