package diego

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	. "github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers"
)

var _ = Describe("An application being staged with Diego", func() {
	var appName string

	BeforeEach(func() {
		appName = RandomName()
	})

	AfterEach(func() {
		Eventually(Cf("delete", appName, "-f"), DefaultTimeout).Should(Exit(0))
	})

	It("has its staging log streamed during a push", func() {
		//Diego needs a custom buildpack until the ruby buildpack lands
		Eventually(Cf("push", appName, "-p", NewAssets().Dora, "--no-start", "-b=https://github.com/cloudfoundry/cf-buildpack-ruby/archive/master.zip"), CFPushTimeout).Should(Exit(0))
		Eventually(Cf("set-env", appName, "CF_DIEGO_BETA", "true"), DefaultTimeout).Should(Exit(0))

		start := Cf("start", appName)

		Eventually(start, CFPushTimeout).Should(Say("Downloading App Package"))
		Eventually(start, CFPushTimeout).Should(Say("Downloaded App Package"))
		Eventually(start, CFPushTimeout).Should(Say(`Staging\.\.\.`))
		Eventually(start, CFPushTimeout).Should(Exit(0))
	})
})
