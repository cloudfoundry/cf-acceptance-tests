package apps

import (
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/skip_messages"
)

var _ = AppsDescribe("A running application", func() {
	var appName string

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")

		Expect(cf.Cf("push",
			appName,
			"--no-start",
			"-b", Config.RubyBuildpackName,
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", assets.NewAssets().Dora,
			"-d", Config.AppsDomain).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
		app_helpers.SetBackend(appName)
		Expect(cf.Cf("start", appName).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
	})

	AfterEach(func() {
		app_helpers.AppReport(appName, DEFAULT_TIMEOUT)

		Expect(cf.Cf("delete", appName, "-f", "-r").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	})

	It("can have its files inspected", func() {
		// Currently cannot work with multiple instances since CF always checks instance 0
		if Config.Backend != "dea" {
			Skip(skip_messages.SkipDeaMessage)
		}
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

	It("shows crash events and recovers from crashes", func() {
		id := helpers.CurlApp(appName, "/id")
		helpers.CurlApp(appName, "/sigterm/KILL")

		Eventually(func() string {
			return string(cf.Cf("events", appName).Wait(DEFAULT_TIMEOUT).Out.Contents())
		}, DEFAULT_TIMEOUT).Should(MatchRegexp("[eE]xited"))

		Eventually(func() string { return helpers.CurlApp(appName, "/id") }).Should(Not(Equal(id)))
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
