package apps

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
)

var _ = Describe("A running application", func() {
	var appName string

	BeforeEach(func() {
		appName = generator.PrefixedRandomName("CATS-APP-")

		Expect(cf.Cf("push", appName, "-m", "128M", "-p", assets.NewAssets().Dora, "-d", config.AppsDomain).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
	})

	AfterEach(func() {
		Expect(cf.Cf("delete", appName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	})

	It(diegoUnsupportedTag+"can have its files inspected", func() {
		// Currently cannot work with multiple instances since CF always checks instance 0
		files := cf.Cf("files", appName).Wait(DEFAULT_TIMEOUT)
		Expect(files).To(Exit(0))
		Expect(files).To(Say("app/"))

		files = cf.Cf("files", appName, "app/").Wait(DEFAULT_TIMEOUT)
		Expect(files).To(Exit(0))
		Expect(files).To(Say("config.ru"))

		files = cf.Cf("files", appName, "app/config.ru").Wait(DEFAULT_TIMEOUT)
		Expect(files).To(Exit(0))
		Expect(files).To(Say("run Dora"))
	})

	It("can show crash events", func() {
		helpers.CurlApp(appName, "/sigterm/KILL")

		Eventually(func() string {
			return string(cf.Cf("events", appName).Wait(DEFAULT_TIMEOUT).Out.Contents())
		}, DEFAULT_TIMEOUT).Should(MatchRegexp("[eE]xited"))
	})

	Context("with multiple instances", func() {
		BeforeEach(func() {
			Expect(cf.Cf("scale", appName, "-i", "2").Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
		})

		It("can be queried for state by instance", func() {
			app := cf.Cf("app", appName).Wait(DEFAULT_TIMEOUT)
			Expect(app).To(Exit(0))
			Expect(app).To(Say("#0"))
			Expect(app).To(Say("#1"))
		})
	})
})
