package diego

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers"
)

var _ = Describe("When staging fails", func() {
	var appName string

	BeforeEach(func() {
		appName = generator.RandomName()

		//Diego needs a custom buildpack until the ruby buildpack lands
		Eventually(cf.Cf("push", appName, "-p", helpers.NewAssets().Dora, "--no-start", "-b=http://example.com/so-not-a-thing/adlfijaskldjlkjaslbnalwieulfjkjsvas.zip"), CF_PUSH_TIMEOUT).Should(Exit(0))
		Eventually(cf.Cf("set-env", appName, "CF_DIEGO_BETA", "true"), DEFAULT_TIMEOUT).Should(Exit(0))
	})

	AfterEach(func() {
		Eventually(cf.Cf("delete", appName, "-f"), DEFAULT_TIMEOUT).Should(Exit(0))
	})

	It("informs the user", func() {
		start := cf.Cf("start", appName)
		Eventually(start, CF_PUSH_TIMEOUT).Should(Exit(1))
		Ω(start.Out).Should(Say("Failed to Download Buildpack"))
	})
})
