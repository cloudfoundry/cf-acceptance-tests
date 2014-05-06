package diego

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	. "github.com/cloudfoundry/cf-acceptance-tests/helpers"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/generator"
)

var _ = Describe("When staging fails", func() {
	var appName string

	BeforeEach(func() {
		appName = RandomName()

		//Diego needs a custom buildpack until the ruby buildpack lands
		Eventually(Cf("push", appName, "-p", NewAssets().Dora, "--no-start", "-b=http://example.com/so-not-a-thing/adlfijaskldjlkjaslbnalwieulfjkjsvas.zip"), CFPushTimeout).Should(Exit(0))
		Eventually(Cf("set-env", appName, "CF_DIEGO_BETA", "true"), DefaultTimeout).Should(Exit(0))
	})

	AfterEach(func() {
		Eventually(Cf("delete", appName, "-f"), DefaultTimeout).Should(Exit(0))
	})

	It("informs the user", func() {
		start := Cf("start", appName)
		Eventually(start, CFPushTimeout).Should(Exit(1))

		//this fails so fast that the CLI can't stream the logging output, so we grab the recent logs instead
		logs := Cf("logs", appName, "--recent").Wait(DefaultTimeout).Out.Contents()
		Ω(logs).Should(ContainSubstring("Failed to Download Buildpack"))
	})
})
