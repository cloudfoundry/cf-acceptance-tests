package lifecycle

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/vito/cmdtest/matchers"

	. "github.com/vito/runtime-integration/helpers"
)

var _ = Describe("Application", func() {
	BeforeEach(func() {
		AppName = RandomName()

		Expect(Cf("push", AppName, "-p", doraPath)).To(
			SayWithTimeout("Started", 2*time.Minute),
		)
	})

	AfterEach(func() {
		Expect(Cf("delete", AppName, "-f")).To(
			SayWithTimeout("OK", 30*time.Second),
		)
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
				Expect(Cf("start", AppName)).To(
					SayWithTimeout("Started", 30*time.Second),
				)
			})

			It("makes the app reachable again", func() {
				Eventually(Curling("/")).Should(Say("Hi, I'm Dora!"))
			})
		})
	})

	Describe("updating", func() {
		It("is reflected through another push", func() {
			Eventually(Curling("/")).Should(Say("Hi, I'm Dora!"))

			Expect(Cf("push", AppName, "-p", helloPath)).To(
				SayWithTimeout("Started", 2*time.Minute),
			)

			Eventually(Curling("/")).Should(Say("Hello, world!"))
		})
	})

	Describe("deleting", func() {
		BeforeEach(func() {
			Expect(Cf("delete", AppName, "-f")).To(
				SayWithTimeout("OK", 30*time.Second),
			)
		})

		It("removes the application", func() {
			Expect(Cf("app", AppName)).To(Say("not found"))
		})

		It("makes the app unreachable", func() {
			Eventually(Curling("/")).Should(Say("404"))
		})
	})
})
