package apps

import (
	"encoding/json"
	"fmt"
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
	"time"
)

var _ = AppsDescribe("Crashing", func() {
	var appName string

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)
		Expect(cf.Cf("delete", appName, "-f", "-r").Wait()).To(Exit(0))
	})

	Describe("a continuously crashing app", func() {
		It("emits crash events and reports as 'crashed' after enough crashes", func() {
			Expect(cf.Cf(
				"push",
				appName,
				"-c", "/bin/false",
				"-b", Config.GetBinaryBuildpackName(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", assets.NewAssets().Catnip,
			).Wait(Config.CfPushTimeoutDuration())).To(Exit(1))

			Eventually(func() string {
				return string(cf.Cf("events", appName).Wait().Out.Contents())
			}).Should(MatchRegexp("app.crash"))

			Eventually(cf.Cf("app", appName)).Should(Say("crashed"))
		})
	})

	Describe("an app with three instances, two running and one crashing", func() {
		It("keeps two instances running while another crashes", func() {
			By("Pushing the app with three instances")
			Expect(cf.Cf(
				"push", appName,
				"-b", "python_buildpack",
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", assets.NewAssets().PythonCrashApp,
				"-i", "3", // Setting three instances
			).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			By("Checking that the app is up and running")
			Eventually(cf.Cf("app", appName)).Should(Say("running"))

			By("Waiting until one instance crashes")
			appGuid := app_helpers.GetAppGuid(appName)
			processStatsPath := fmt.Sprintf("/v3/apps/%s/processes/web/stats", appGuid)

			// Poll until at least one instance has crashed
			Eventually(func() bool {
				session := cf.Cf("curl", processStatsPath).Wait()
				instancesJson := struct {
					Resources []struct {
						Type  string `json:"type"`
						State string `json:"state"`
					} `json:"resources"`
				}{}

				bytes := session.Wait().Out.Contents()
				err := json.Unmarshal(bytes, &instancesJson)
				Expect(err).ToNot(HaveOccurred())

				for _, instance := range instancesJson.Resources {
					if instance.State == "CRASHED" {
						return true
					}
				}
				return false
			}, 60*time.Second, 5*time.Second).Should(BeTrue(), "At least one instance should be in the CRASHED state")

			By("Verifying at least one instance is still running")
			session := cf.Cf("curl", processStatsPath).Wait()
			instancesJson := struct {
				Resources []struct {
					Type  string `json:"type"`
					State string `json:"state"`
				} `json:"resources"`
			}{}

			bytes := session.Wait().Out.Contents()
			json.Unmarshal(bytes, &instancesJson)

			foundRunning := false
			for _, instance := range instancesJson.Resources {
				if instance.State == "RUNNING" {
					foundRunning = true
					break
				}
			}
			Expect(foundRunning).To(BeTrue(), "At least one instance should still be in the RUNNING state")
		})
	})

	Context("the app crashes", func() {
		BeforeEach(func() {
			Expect(cf.Cf(app_helpers.CatnipWithArgs(
				appName,
				"-m", DEFAULT_MEMORY_LIMIT)...,
			).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		})

		It("shows crash events", func() {
			helpers.CurlApp(Config, appName, "/sigterm/KILL")
			Eventually(func() string {
				return string(cf.Cf("events", appName).Wait().Out.Contents())
			}).Should(MatchRegexp("app.crash"))
		})

		It("recovers", func() {
			const idChecker = "^[0-9a-zA-Z]+(?:-[0-9a-zA-z]+)+$"

			id := helpers.CurlApp(Config, appName, "/id")
			Expect(id).Should(MatchRegexp(idChecker))
			helpers.CurlApp(Config, appName, "/sigterm/KILL")

			Eventually(func() string {
				return helpers.CurlApp(Config, appName, "/id")
			}).Should(MatchRegexp(idChecker))
		})
	})
})
