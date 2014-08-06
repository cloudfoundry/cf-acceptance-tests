package apps

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers"
)

type AppUsageEvent struct {
	Entity struct {
		AppName       string `json:"app_name"`
		State         string `json:"state"`
		BuildpackName string `json:"buildpack_name"`
		BuildpackGuid string `json:"buildpack_guid"`
	} `json:"entity"`
}

type AppUsageEvents struct {
	Resources []AppUsageEvent `struct:"resources"`
}

func lastAppUsageEvent(appName string, state string) (bool, AppUsageEvent) {
	var response AppUsageEvents
	cf.AsUser(context.AdminUserContext(), func() {
		cf.ApiRequest("GET", "/v2/app_usage_events?order-direction=desc&page=1", &response)
	})

	for _, event := range response.Resources {
		if event.Entity.AppName == appName && event.Entity.State == state {
			return true, event
		}
	}

	return false, AppUsageEvent{}
}

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
			Eventually(func() string {
				return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hi, I'm Dora!"))
		})

		It("generates an app usage 'started' event", func() {
			found, _ := lastAppUsageEvent(appName, "STARTED")
			Expect(found).To(BeTrue())
		})

		It("generates an app usage 'buildpack_set' event", func() {
			found, matchingEvent := lastAppUsageEvent(appName, "BUILDPACK_SET")

			Expect(found).To(BeTrue())
			Expect(matchingEvent.Entity.BuildpackName).To(Equal("ruby_buildpack"))
			Expect(matchingEvent.Entity.BuildpackGuid).ToNot(BeZero())
		})
	})

	Describe("stopping", func() {
		BeforeEach(func() {
			Expect(cf.Cf("stop", appName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		})

		It("makes the app unreachable", func() {
			Eventually(func() string {
				return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("404"))
		})

		It("generates an app usage 'stopped' event", func() {
			found, _ := lastAppUsageEvent(appName, "STOPPED")
			Expect(found).To(BeTrue())
		})

		Describe("and then starting", func() {
			BeforeEach(func() {
				Expect(cf.Cf("start", appName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			})

			It("makes the app reachable again", func() {
				Eventually(func() string {
					return helpers.CurlAppRoot(appName)
				}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hi, I'm Dora!"))
			})
		})
	})

	Describe("updating", func() {
		It("is reflected through another push", func() {
			Eventually(func() string {
				return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hi, I'm Dora!"))

			Expect(cf.Cf("push", appName, "-p", helpers.NewAssets().HelloWorld).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hello, world!"))
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
			Eventually(func() string {
				return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("404"))
		})

		It("generates an app usage 'stopped' event", func() {
			found, _ := lastAppUsageEvent(appName, "STOPPED")
			Expect(found).To(BeTrue())
		})
	})
})
