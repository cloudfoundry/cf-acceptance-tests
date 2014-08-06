package apps

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers"
)

var _ = Describe("Buildpacks", func() {
	var appName string

	BeforeEach(func() {
		appName = generator.RandomName()
	})

	AfterEach(func() {
		Expect(cf.Cf("delete", appName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	})

	Describe("node", func() {
		It("makes the app reachable via its bound route", func() {
			Expect(cf.Cf("push", appName, "-p", helpers.NewAssets().Node, "-c", "node app.js").Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hello from a node app!"))
		})
	})

	Describe("java", func() {
		It("makes the app reachable via its bound route", func() {
			Expect(cf.Cf("push", appName, "-p", helpers.NewAssets().Java).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hello, from your friendly neighborhood Java JSP!"))
		})
	})

	Describe("golang", func() {
		It("makes the app reachable via its bound route", func() {
			Expect(cf.Cf("push", appName, "-p", helpers.NewAssets().Golang).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("go, world"))
		})
	})

	Describe("python", func() {
		It("makes the app reachable via its bound route", func() {
			Expect(cf.Cf("push", appName, "-p", helpers.NewAssets().Python).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("python, world"))
		})
	})

	Describe("php", func() {
		// This test requires more time during push, because the php buildpack is slower than your average bear
		var phpPushTimeout = CF_PUSH_TIMEOUT + 6*time.Minute

		It("makes the app reachable via its bound route", func() {
			Expect(cf.Cf("push", appName, "-p", helpers.NewAssets().Php).Wait(phpPushTimeout)).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hello from php"))
		})
	})
})
