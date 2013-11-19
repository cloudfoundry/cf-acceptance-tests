package apps

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/vito/cmdtest/matchers"

	. "github.com/pivotal-cf-experimental/cf-acceptance-tests/helpers"
)

var _ = Describe("Application", func() {
	BeforeEach(func() {
		AppName = RandomName()

		Expect(Cf("push", AppName, "-p", doraPath)).To(Say("Started"))
	})

	AfterEach(func() {
		Expect(Cf("delete", AppName, "-f")).To(Say("OK"))
	})

	Describe("pushing", func() {
		It("makes the app reachable via its bound route", func() {
			Eventually(Curling("/")).Should(Say("Hi, I'm Dora!"))
		})
	})

	Describe("stopping", func() {
		BeforeEach(func() {
			Expect(Cf("stop", AppName)).To(Say("OK"))
		})

		It("makes the app unreachable", func() {
			Eventually(Curling("/"), 5.0).Should(Say("404"))
		})

		Describe("and then starting", func() {
			BeforeEach(func() {
				Expect(Cf("start", AppName)).To(Say("Started"))
			})

			It("makes the app reachable again", func() {
				Eventually(Curling("/")).Should(Say("Hi, I'm Dora!"))
			})
		})
	})

	Describe("updating", func() {
		It("is reflected through another push", func() {
			Eventually(Curling("/")).Should(Say("Hi, I'm Dora!"))

			Expect(Cf("push", AppName, "-p", helloPath)).To(Say("Started"))

			Eventually(Curling("/")).Should(Say("Hello, world!"))
		})
	})

	Describe("deleting", func() {
		BeforeEach(func() {
			Expect(Cf("delete", AppName, "-f")).To(Say("OK"))
		})

		It("removes the application", func() {
			Expect(Cf("app", AppName)).To(Say("not found"))
		})

		It("makes the app unreachable", func() {
			Eventually(Curling("/")).Should(Say("404"))
		})
	})
})
