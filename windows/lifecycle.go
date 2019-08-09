package windows

import (
	"regexp"
	"strconv"
	"time"

	. "github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
)

var _ = WindowsDescribe("Application Lifecycle", func() {
	var appName string

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")
	})

	It("exercises the app through its lifecycle", func() {
		By("pushing it", func() {
			Expect(cf.Push(appName,
				"-s", Config.GetWindowsStack(),
				"-b", Config.GetHwcBuildpackName(),
				"-m", DEFAULT_WINDOWS_MEMORY_LIMIT,
				"-p", assets.NewAssets().Nora,
				"-d", Config.GetAppsDomain()).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		})

		By("checking the 'started' event", func() {
			found, _ := lastAppUsageEvent(appName, "STARTED")
			Expect(found).To(BeTrue())
		})

		By("verifying it's up", func() {
			Eventually(helpers.CurlingAppRoot(Config, appName)).Should(ContainSubstring("hello i am nora"))
		})

		By("verifying reported disk/memory usage", func() {
			// #0   running   2015-06-10 02:22:39 PM   0.0%   48.7M of 2G   14M of 1G
			var metrics = regexp.MustCompile(`running.*(?:[\d\.]+)%\s+([\d\.]+)[MG]? of (?:[\d\.]+)[MG]\s+([\d\.]+)[MG]? of (?:[\d\.]+)[MG]`)
			memdisk := func() (float64, float64) {
				app := cf.Cf("app", appName)
				Expect(app.Wait()).To(Exit(0))

				arr := metrics.FindStringSubmatch(string(app.Out.Contents()))
				Expect(arr).NotTo(BeNil())

				mem, err := strconv.ParseFloat(arr[1], 64)
				Expect(err).ToNot(HaveOccurred())
				disk, err := strconv.ParseFloat(arr[2], 64)
				Expect(err).ToNot(HaveOccurred())
				return mem, disk
			}
			Eventually(func() float64 { m, _ := memdisk(); return m }, Config.CfPushTimeoutDuration()).Should(BeNumerically(">", 0.0))
			Eventually(func() float64 { _, d := memdisk(); return d }, Config.CfPushTimeoutDuration()).Should(BeNumerically(">", 0.0))
		})

		By("makes system environment variables available", func() {
			Eventually(func() string {
				return helpers.CurlApp(Config, appName, "/env")
			}).Should(ContainSubstring(`"INSTANCE_GUID"`))
		})

		By("stopping it", func() {
			Expect(cf.Cf("stop", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
			Eventually(helpers.CurlingAppRoot(Config, appName)).Should(ContainSubstring("404"))
		})

		By("setting an environment variable", func() {
			Expect(cf.Cf("set-env", appName, "FOO", "bar").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		})

		By("starting it", func() {
			Expect(cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
			Eventually(helpers.CurlingAppRoot(Config, appName)).Should(ContainSubstring("hello i am nora"))
		})

		By("checking custom env variables are available", func() {
			Eventually(func() string {
				return helpers.CurlAppWithTimeout(Config, appName, "/env/FOO", 30*time.Second)
			}).Should(ContainSubstring("bar"))
		})

		By("scaling it", func() {
			Expect(cf.Cf("scale", appName, "-i", "2").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
			Eventually(cf.Cf("apps")).Should(gbytes.Say("2/2"))
			Expect(cf.Cf("app", appName).Wait()).ToNot(gbytes.Say("insufficient resources"))
		})

		By("restarting an instance", func() {
			idsBefore := app_helpers.ReportedIDs(2, appName)
			Expect(len(idsBefore)).To(Equal(2))
			Expect(cf.Cf("restart-app-instance", appName, "1").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
			Eventually(func() []string {
				return app_helpers.DifferentIDsFrom(idsBefore, appName)
			}, Config.CfPushTimeoutDuration(), 2*time.Second).Should(HaveLen(1))
		})

		By("updating, is reflected through another push", func() {
			Eventually(helpers.CurlingAppRoot(Config, appName)).Should(ContainSubstring("hello i am nora"))

			Expect(cf.Cf(
				"push", appName,
				"-p", assets.NewAssets().WindowsWebapp,
				"-c", ".\\webapp.exe",
				"-b", Config.GetBinaryBuildpackName(),
			).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			Eventually(helpers.CurlingAppRoot(Config, appName)).Should(ContainSubstring("hi i am a standalone webapp"))
		})

		By("removing it", func() {
			Expect(cf.Cf("delete", appName, "-f").Wait()).To(Exit(0))
			app := cf.Cf("app", appName).Wait()
			Expect(app).To(Exit(1))
			Expect(app.Err).To(gbytes.Say("not found"))

			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}).Should(ContainSubstring("404"))

			found, _ := lastAppUsageEvent(appName, "STOPPED")
			Expect(found).To(BeTrue())
		})
	})
})

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
	AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
		ApiRequest("GET", "/v2/app_usage_events?order-direction=desc&page=1&results-per-page=150", &response, Config.DefaultTimeoutDuration())
	})

	for _, event := range response.Resources {
		if event.Entity.AppName == appName && event.Entity.State == state {
			return true, event
		}
	}

	return false, AppUsageEvent{}
}
