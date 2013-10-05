package lifecycle

import (
	"time"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/vito/cmdtest/matchers"

	"github.com/vito/runtime-integration/config"
	. "github.com/vito/runtime-integration/helpers"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Lifecycle")
}

var conf = config.Load()
var appName = RandomName()
var appPath = "../assets/dora"
var appUri = "http://" + appName + "." + conf.AppsDomain

var _ = Describe("Application", func() {
	Describe("pushing", func() {
		It("works", func() {
			push := Run("go-cf", "push", appName, "-p", appPath)
			Expect(push).To(SayWithTimeout("Started", 2 * time.Minute))
			Expect(push).To(ExitWith(0))
		})

		It("makes the app reachable via its bound routes", func() {
			curl := Run("curl", "-s", appUri)
			Expect(curl).To(Say("Hello, world!"))
			Expect(curl).To(ExitWith(0))
		})
	})

	Describe("stopping", func() {
		It("works", func() {
			del := Run("go-cf", "stop", appName)
			Expect(del).To(Say("OK"))
			Expect(del).To(ExitWith(0))
		})

		It("makes the app unreachable", func() {
			curl := Run("curl", "-s", appUri)
			Expect(curl).To(Say("404"))
			Expect(curl).To(ExitWith(0))
		})
	})

	Describe("starting", func() {
		It("works", func() {
			del := Run("go-cf", "start", appName)
			Expect(del).To(Say("OK"))
			Expect(del).To(ExitWith(0))
		})

		It("makes the app reachable again", func() {
			curl := Run("curl", "-s", appUri)
			Expect(curl).To(Say("Hello, world!"))
			Expect(curl).To(ExitWith(0))
		})
	})

	Describe("deleting", func() {
		It("works", func() {
			del := Run("go-cf", "delete", appName, "-f")
			Expect(del).To(Say("OK"))
			Expect(del).To(ExitWith(0))
		})

		It("removes the application", func() {
			app := Run("go-cf", "app", appName)
			Expect(app).To(Say("not found"))
			Expect(app).To(ExitWith(1))
		})

		It("makes the app unreachable", func() {
			curl := Run("curl", "-s", appUri)
			Expect(curl).To(Say("404"))
			Expect(curl).To(ExitWith(0))
		})
	})
})
