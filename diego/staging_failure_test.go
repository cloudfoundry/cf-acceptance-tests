package diego

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("When staging fails", func() {
	var appName string
	var start *Session

	BeforeEach(func() {
		appName = generator.RandomName()

		//Diego needs a custom buildpack until the ruby buildpack lands
		Eventually(cf.Cf("push", appName, "-p", assets.NewAssets().Dora, "--no-start", "-b=http://example.com/so-not-a-thing/adlfijaskldjlkjaslbnalwieulfjkjsvas.zip"), CF_PUSH_TIMEOUT).Should(Exit(0))
		Eventually(cf.Cf("set-env", appName, "CF_DIEGO_BETA", "true"), DEFAULT_TIMEOUT).Should(Exit(0))
	})

	AfterEach(func() {
		Eventually(cf.Cf("delete", appName, "-f"), DEFAULT_TIMEOUT).Should(Exit(0))
	})

	JustBeforeEach(func() {
		start = cf.Cf("start", appName)
	})

	It("informs the user in the CLI output and the logs", func() {
		Eventually(start, CF_PUSH_TIMEOUT).Should(Exit(1))
		Ω(start.Out).Should(gbytes.Say("Staging error: cannot get instances since staging failed"))

		Eventually(func() *Session {
			logs := cf.Cf("logs", appName, "--recent")
			Expect(logs.Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			return logs
		}, DEFAULT_TIMEOUT).Should(gbytes.Say("Failed to Download Buildpack"))
	})
})
