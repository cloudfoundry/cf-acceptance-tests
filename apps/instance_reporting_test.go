package apps

import (
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Getting instance information", func() {
	var appName string

	BeforeEach(func() {
		appName = generator.RandomName()

		Expect(cf.Cf("push", appName, "-p", assets.NewAssets().HelloWorld,).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

		Eventually(cf.Cf("scale", appName, "-i", "3")).Should(Exit(0))
	})

	AfterEach(func() {
		Eventually(cf.Cf("delete", appName, "-f")).Should(Exit(0))
	})

	It("Retrieves instance information for cf app and cf apps", func() {
		By("calling cf app")
		app := cf.Cf("app", appName).Wait(DEFAULT_TIMEOUT)
		Expect(app).To(Exit(0))
		Expect(app).To(Say("instances: [0-3]/3"))
		Expect(app).To(Say("#0"))
		Expect(app).To(Say("#1"))
		Expect(app).To(Say("#2"))
		Expect(app).ToNot(Say("#3"))

		By("calling cf apps")
		app = cf.Cf("apps").Wait(DEFAULT_TIMEOUT)
		Expect(app).To(Exit(0))
		Expect(app).To(Say(appName))
		Expect(app).To(Say("[0-3]/3"))
	})
})

