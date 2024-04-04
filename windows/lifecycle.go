package windows

import (
	"regexp"
	"strconv"
	"time"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/helpers"
)

var _ = WindowsDescribe("Application Lifecycle", func() {
	var appName string

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")
	})

	It("exercises the app through its lifecycle", func() {
		By("pushing it", func() {
			Expect(cf.Cf("push",
				appName,
				"-s", Config.GetWindowsStack(),
				"-b", Config.GetHwcBuildpackName(),
				"-m", DEFAULT_WINDOWS_MEMORY_LIMIT,
				"-p", assets.NewAssets().Nora,
			).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		})

		By("checking the 'started' event", func() {
			found, _ := app_helpers.LastAppUsageEventByState(appName, "STARTED")
			Expect(found).To(BeTrue())
		})

		By("verifying it's up", func() {
			Eventually(helpers.CurlingAppRoot(Config, appName)).Should(ContainSubstring("hello i am nora"))
		})

		By("verifying reported usage", func() {
			// #0   running   2015-06-10 02:22:39 PM   0.0%   48.7M of 2G   14M of 1G     68B/s of unlimited
			var metrics = regexp.MustCompile(`running.*(?:[\d\.]+)%\s+([\d\.]+)[BKMG]? of (?:[\d\.]+)[BKMG]?\s+([\d\.]+)[BKMG]? of (?:[\d\.]+)[BKMG]?\s+([\d\.]+)[BKMG]?/s of (?:[\d\.]+[BKMG]?/s|unlimited)`)
			stats := func() (float64, float64, float64) {
				helpers.CurlApp(Config, appName, "/logspew/1024")

				app := cf.Cf("app", appName)
				Expect(app.Wait()).To(Exit(0))

				arr := metrics.FindStringSubmatch(string(app.Out.Contents()))
				Expect(arr).NotTo(BeNil())

				mem, err := strconv.ParseFloat(arr[1], 64)
				Expect(err).ToNot(HaveOccurred())
				disk, err := strconv.ParseFloat(arr[2], 64)
				Expect(err).ToNot(HaveOccurred())
				logs, err := strconv.ParseFloat(arr[3], 64)
				Expect(err).ToNot(HaveOccurred())
				return mem, disk, logs
			}
			Eventually(func() float64 { m, _, _ := stats(); return m }, Config.CfPushTimeoutDuration()).Should(BeNumerically(">", 0.0))
			Eventually(func() float64 { _, d, _ := stats(); return d }, Config.CfPushTimeoutDuration()).Should(BeNumerically(">", 0.0))
			Eventually(func() float64 { _, _, l := stats(); return l }, Config.CfPushTimeoutDuration()).Should(BeNumerically(">", 0.0))
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
			time.Sleep(5 * time.Second)
			Eventually(func() string {
				apps := cf.Cf("apps")
				Expect(apps.Wait()).To(Exit(0))
				return string(apps.Out.Contents())
			}, Config.DefaultTimeoutDuration()).Should(ContainSubstring("web:2/2"))
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

			found, _ := app_helpers.LastAppUsageEventByState(appName, "STOPPED")
			Expect(found).To(BeTrue())
		})
	})
})
