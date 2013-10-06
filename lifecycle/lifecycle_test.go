package lifecycle

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vito/cmdtest"
	. "github.com/vito/cmdtest/matchers"

	"github.com/vito/runtime-integration/config"
	. "github.com/vito/runtime-integration/helpers"
)

func TestLifecycle(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Application Lifecycle")
}

var conf = config.Load()

var appName = ""
var doraPath = "../assets/dora"
var helloPath = "../assets/hello-world"

func appUri() string {
	return "http://" + appName + "." + conf.AppsDomain
}

func Curling(endpoint string) func() *cmdtest.Session {
	return func() *cmdtest.Session {
		return Curl(appUri() + endpoint)
	}
}

var _ = Describe("Application", func() {
	BeforeEach(func() {
		appName = RandomName()

		Expect(Cf("push", appName, "-p", doraPath)).To(
			SayWithTimeout("Started", 2*time.Minute),
		)
	})

	AfterEach(func() {
		Expect(Cf("delete", appName, "-f")).To(Say("OK"))
	})

	Describe("pushing", func() {
		It("makes the app reachable via its bound route", func() {
			Eventually(Curling("/")).Should(Say("Hi, I'm Dora!"))
		})
	})

	Describe("stopping", func() {
		BeforeEach(func() {
			Expect(Cf("stop", appName)).To(Say("OK"))
		})

		It("makes the app unreachable", func() {
			Eventually(Curling("/"), 5.0).Should(Say("404"))
		})

		Describe("and then starting", func() {
			BeforeEach(func() {
				Expect(Cf("start", appName)).To(Say("OK"))
			})

			It("makes the app reachable again", func() {
				Eventually(Curling("/"), 10.0).Should(Say("Hi, I'm Dora!"))
			})
		})
	})

	Describe("updating", func() {
		It("is reflected through another push", func() {
			Eventually(Curling("/")).Should(Say("Hi, I'm Dora!"))

			Expect(Cf("push", appName, "-p", helloPath)).To(
				SayWithTimeout("Started", 2*time.Minute),
			)

			Eventually(Curling("/")).Should(Say("Hello, world!"))
		})
	})

	Describe("deleting", func() {
		BeforeEach(func() {
			Expect(Cf("delete", appName, "-f")).To(Say("OK"))
		})

		It("removes the application", func() {
			Expect(Cf("app", appName)).To(Say("not found"))
		})

		It("makes the app unreachable", func() {
			Eventually(Curling("/")).Should(Say("404"))
		})
	})
})
