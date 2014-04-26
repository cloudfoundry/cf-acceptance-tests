package apps

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

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
		Eventually(Cf("delete", appName, "-f"), DefaultTimeout).Should(Exit(0))
	})

	It("has its staging log streamed during a push", func() {
		push := Cf("push", appName, "-p", NewAssets().Dora)

		Eventually(push, CFPushTimeout).Should(Say("Installing dependencies"))
		Eventually(push, CFPushTimeout).Should(Say("Uploading droplet"))
		Eventually(push, CFPushTimeout).Should(Say("App started"))
		Eventually(push, CFPushTimeout).Should(Exit(0))
	})
})
