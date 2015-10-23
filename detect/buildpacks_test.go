package detect

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
)

var _ = Describe("Buildpacks", func() {
	var appName string

	BeforeEach(func() {
		appName = generator.PrefixedRandomName("CATS-APP-")
	})

	AfterEach(func() {
		Eventually(cf.Cf("logs", appName, "--recent"), DEFAULT_TIMEOUT).Should(Exit())
		Expect(cf.Cf("delete", appName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	})

	Describe("node", func() {
		It("makes the app reachable via its bound route", func() {
			Expect(cf.Cf("push", appName, "--no-start", "-m", "128M", "-p", assets.NewAssets().Node, "-d", config.AppsDomain).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			app_helpers.ConditionallyEnableDiego(appName)
			Expect(cf.Cf("start", appName).Wait(DETECT_TIMEOUT)).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hello from a node app!"))
		})
	})

	Describe("java", func() {
		It("makes the app reachable via its bound route", func() {
			Expect(cf.Cf("push", appName, "--no-start", "-p", assets.NewAssets().Java, "-m", "512M", "-d", config.AppsDomain).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			app_helpers.ConditionallyEnableDiego(appName)
			Expect(cf.Cf("set-env", appName, "JAVA_OPTS", "-Djava.security.egd=file:///dev/urandom").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			Expect(cf.Cf("start", appName).Wait(CF_JAVA_TIMEOUT)).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hello, from your friendly neighborhood Java JSP!"))
		})
	})

	Describe("golang", func() {
		It("makes the app reachable via its bound route", func() {
			Expect(cf.Cf("push", appName, "--no-start", "-m", "128M", "-p", assets.NewAssets().Golang, "-d", config.AppsDomain).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			app_helpers.ConditionallyEnableDiego(appName)
			Expect(cf.Cf("start", appName).Wait(DETECT_TIMEOUT)).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("go, world"))
		})
	})

	Describe("python", func() {
		It("makes the app reachable via its bound route", func() {
			Expect(cf.Cf("push", appName, "--no-start", "-m", "128M", "-p", assets.NewAssets().Python, "-d", config.AppsDomain).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			app_helpers.ConditionallyEnableDiego(appName)
			Expect(cf.Cf("start", appName).Wait(DETECT_TIMEOUT)).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("python, world"))
		})
	})

	Describe("php", func() {
		// This test requires more time during push, because the php buildpack is slower than your average bear
		var phpPushTimeout = DETECT_TIMEOUT + 6*time.Minute

		It("makes the app reachable via its bound route", func() {
			Expect(cf.Cf("push", appName, "--no-start", "-m", "128M", "-p", assets.NewAssets().Php, "-d", config.AppsDomain).Wait(phpPushTimeout)).To(Exit(0))
			app_helpers.ConditionallyEnableDiego(appName)
			Expect(cf.Cf("start", appName).Wait(DETECT_TIMEOUT)).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hello from php"))
		})
	})

	Describe("staticfile", func() {
		It("makes the app reachable via its bound route", func() {
			Expect(cf.Cf("push", appName, "--no-start", "-m", "128M", "-p", assets.NewAssets().Staticfile, "-d", config.AppsDomain).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			app_helpers.ConditionallyEnableDiego(appName)
			Expect(cf.Cf("start", appName).Wait(DETECT_TIMEOUT)).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hello from a staticfile"))
		})
	})

	Describe("binary", func() {
		It("makes the app reachable via its bound route", func() {
			Expect(cf.Cf("push", appName, "--no-start", "-b", config.BinaryBuildpackName, "-m", "128M", "-p", assets.NewAssets().Binary, "-d", config.AppsDomain).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			app_helpers.ConditionallyEnableDiego(appName)
			Expect(cf.Cf("start", appName).Wait(DETECT_TIMEOUT)).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hello from a binary"))
		})
	})
})
