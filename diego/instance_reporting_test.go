package diego

import (
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Getting instance information", func() {
	var appName string

	BeforeEach(func() {
		appName = generator.RandomName()

		Eventually(cf.Cf("push", appName, "-p", helpers.NewAssets().HelloWorld, "--no-start", "-b=ruby_buildpack"), CF_PUSH_TIMEOUT).Should(Exit(0))
		Eventually(cf.Cf("set-env", appName, "CF_DIEGO_RUN_BETA", "true"), DEFAULT_TIMEOUT).Should(Exit(0))

		Eventually(cf.Cf("scale", appName, "-i", "3"), DEFAULT_TIMEOUT).Should(Exit(0))
		Eventually(cf.Cf("start", appName), CF_PUSH_TIMEOUT).Should(Exit(0))
	})

	AfterEach(func() {
		Eventually(cf.Cf("delete", appName, "-f"), DEFAULT_TIMEOUT).Should(Exit(0))
	})

	It("Retrieves instance information for cf app", func() {
		app := cf.Cf("app", appName).Wait(DEFAULT_TIMEOUT)
		Expect(app).To(Exit(0))
		Expect(app).To(Say("instances: [0-3]/3"))
		Expect(app).To(Say("#0"))
		Expect(app).To(Say("#1"))
		Expect(app).To(Say("#2"))
		Expect(app).ToNot(Say("#3"))
	})

	It("Retrieves instance information for cf apps", func() {
		app := cf.Cf("apps").Wait(DEFAULT_TIMEOUT)
		Expect(app).To(Exit(0))
		Expect(app).To(Say(appName))
		Expect(app).To(Say("[0-3]/3"))
	})
})
