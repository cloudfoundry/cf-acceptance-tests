package apps

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers"
	"github.com/pivotal-cf-experimental/cf-test-helpers/cf"
	"github.com/pivotal-cf-experimental/cf-test-helpers/generator"
)

var _ = Describe("Application Lifecycle", func() {
	var appName string

	BeforeEach(func() {
		appName = generator.RandomName()

		Expect(cf.Cf("push", appName, "-p", helpers.NewAssets().Dora).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
	})

	AfterEach(func() {
		Expect(cf.Cf("delete", appName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	})

	Describe("pushing", func() {
		It("makes the app reachable via its bound route", func() {
			Expect(helpers.CurlAppRoot(appName)).To(ContainSubstring("Hi, I'm Dora!"))
		})
	})

	Describe("stopping", func() {
		BeforeEach(func() {
			Expect(cf.Cf("stop", appName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		})

		It("makes the app unreachable", func() {
			Expect(helpers.CurlAppRoot(appName)).To(ContainSubstring("404"))
		})

		Describe("and then starting", func() {
			BeforeEach(func() {
				Expect(cf.Cf("start", appName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			})

			It("makes the app reachable again", func() {
				Expect(helpers.CurlAppRoot(appName)).To(ContainSubstring("Hi, I'm Dora!"))
			})
		})
	})

	Describe("updating", func() {
		It("is reflected through another push", func() {
			Expect(helpers.CurlAppRoot(appName)).To(ContainSubstring("Hi, I'm Dora!"))

			Expect(cf.Cf("push", appName, "-p", helpers.NewAssets().HelloWorld).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

			Expect(helpers.CurlAppRoot(appName)).To(ContainSubstring("Hello, world!"))
		})
	})

	Describe("deleting", func() {
		BeforeEach(func() {
			Expect(cf.Cf("delete", appName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		})

		It("removes the application", func() {
			app := cf.Cf("app", appName).Wait(DEFAULT_TIMEOUT)
			Expect(app).To(Exit(1))
			Expect(app).To(Say("not found"))
		})

		It("makes the app unreachable", func() {
			Expect(helpers.CurlAppRoot(appName)).To(ContainSubstring("404"))
		})
	})
})
