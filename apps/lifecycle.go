package apps

import (
	"encoding/json"
	"fmt"
	"strings"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/skip_messages"
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
	workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
		workflowhelpers.ApiRequest("GET", "/v2/app_usage_events?order-direction=desc&page=1&results-per-page=150", &response, Config.DefaultTimeoutDuration())
	})

	for _, event := range response.Resources {
		if event.Entity.AppName == appName && event.Entity.State == state {
			return true, event
		}
	}

	return false, AppUsageEvent{}
}

var _ = AppsDescribe("Application Lifecycle", func() {
	var (
		appName              string
		expectedNullResponse string
	)

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")

		appUrl := "https://" + appName + "." + Config.GetAppsDomain()
		nullSession := helpers.CurlSkipSSL(Config.GetSkipSSLValidation(), appUrl).Wait(Config.DefaultTimeoutDuration())
		expectedNullResponse = string(nullSession.Buffer().Contents())
	})

	AfterEach(func() {
		app_helpers.AppReport(appName, Config.DefaultTimeoutDuration())

		Expect(cf.Cf("delete", appName, "-f", "-r").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
	})

	Describe("pushing", func() {
		It("makes the app reachable via its bound route", func() {
			Expect(cf.Cf("push",
				appName,
				"--no-start",
				"-b", Config.GetBinaryBuildpackName(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", assets.NewAssets().Catnip,
				"-c", "./catnip",
				"-d", Config.GetAppsDomain()).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
			app_helpers.SetBackend(appName)

			Expect(cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}, Config.DefaultTimeoutDuration()).Should(ContainSubstring("Catnip?"))
		})

		Describe("Context path", func() {
			var app2 string
			var appPath = "/imposter_dora"

			BeforeEach(func() {
				Expect(cf.Cf("push",
					appName,
					"--no-start",
					"-b", Config.GetBinaryBuildpackName(),
					"-m", DEFAULT_MEMORY_LIMIT,
					"-p", assets.NewAssets().Catnip,
					"-c", "./catnip",
					"-d", Config.GetAppsDomain()).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
				app_helpers.SetBackend(appName)

				Expect(cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

				app2 = random_name.CATSRandomName("APP")
				Expect(cf.Cf("push", app2, "--no-start", "-b", Config.GetRubyBuildpackName(), "-m", DEFAULT_MEMORY_LIMIT, "-p", assets.NewAssets().HelloWorld, "-d", Config.GetAppsDomain()).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
				app_helpers.SetBackend(app2)
				Expect(cf.Cf("start", app2).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
			})

			AfterEach(func() {
				Expect(cf.Cf("delete", app2, "-f", "-r").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
			})

			It("makes another app available via same host and domain, but different path", func() {
				getRoutePath := fmt.Sprintf("/v2/routes?q=host:%s", appName)
				routeBody := cf.Cf("curl", getRoutePath).Wait(Config.DefaultTimeoutDuration()).Out.Contents()
				var routeJSON struct {
					Resources []struct {
						Entity struct {
							SpaceGuid  string `json:"space_guid"`
							DomainGuid string `json:"domain_guid"`
						} `json:"entity"`
					} `json:"resources"`
				}
				Expect(json.Unmarshal([]byte(routeBody), &routeJSON)).To(Succeed())

				Expect(len(routeJSON.Resources)).To(BeNumerically(">=", 1))
				spaceGuid := routeJSON.Resources[0].Entity.SpaceGuid
				domainGuid := routeJSON.Resources[0].Entity.DomainGuid
				appGuid := cf.Cf("app", app2, "--guid").Wait(Config.DefaultTimeoutDuration()).Out.Contents()

				jsonBody := "{\"host\":\"" + appName + "\", \"path\":\"" + appPath + "\", \"domain_guid\":\"" + domainGuid + "\",\"space_guid\":\"" + spaceGuid + "\"}"
				routePostResponseBody := cf.Cf("curl", "/v2/routes", "-X", "POST", "-d", jsonBody).Wait(Config.CfPushTimeoutDuration()).Out.Contents()

				var routeResponseJSON struct {
					Metadata struct {
						Guid string `json:"guid"`
					} `json:"metadata"`
				}
				json.Unmarshal([]byte(routePostResponseBody), &routeResponseJSON)
				routeGuid := routeResponseJSON.Metadata.Guid

				Expect(cf.Cf("curl", "/v2/apps/"+strings.TrimSpace(string(appGuid))+"/routes/"+string(routeGuid), "-X", "PUT").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

				Eventually(func() string {
					return helpers.CurlAppRoot(Config, appName)
				}, Config.DefaultTimeoutDuration()).Should(ContainSubstring("Catnip?"))

				Eventually(func() string {
					return helpers.CurlApp(Config, appName, appPath)
				}, Config.DefaultTimeoutDuration()).Should(ContainSubstring("Hello, world!"))
			})
		})

		Context("multiple instances", func() {
			BeforeEach(func() {
				Expect(cf.Cf("push",
					appName,
					"--no-start",
					"-b", Config.GetBinaryBuildpackName(),
					"-m", DEFAULT_MEMORY_LIMIT,
					"-p", assets.NewAssets().Catnip,
					"-c", "./catnip",
					"-d", Config.GetAppsDomain()).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
				app_helpers.SetBackend(appName)
				Expect(cf.Cf("scale", appName, "-i", "2").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
			})

			It("is able to start all instances", func() {
				Expect(cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

				Eventually(func() *Session {
					return cf.Cf("app", appName).Wait(Config.DefaultTimeoutDuration())
				}, Config.DefaultTimeoutDuration()).Should(Say("#0   running"))

				Eventually(func() *Session {
					return cf.Cf("app", appName).Wait(Config.DefaultTimeoutDuration())
				}, Config.DefaultTimeoutDuration()).Should(Say("#1   running"))
			})
		})

		It("makes system environment variables available", func() {
			if Config.GetBackend() != "diego" {
				Skip(skip_messages.SkipDiegoMessage)
			}
			Expect(cf.Cf("push",
				appName,
				"--no-start",
				"-b", Config.GetBinaryBuildpackName(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", assets.NewAssets().Catnip,
				"-c", "./catnip",
				"-d", Config.GetAppsDomain()).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
			app_helpers.SetBackend(appName)

			Expect(cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			var envOutput string
			Eventually(func() string {
				envOutput = helpers.CurlApp(Config, appName, "/env.json")
				return envOutput
			}, Config.DefaultTimeoutDuration()).ShouldNot(Equal(""))
			type env struct {
				Index      string `json:"CF_INSTANCE_INDEX"`
				IP         string `json:"CF_INSTANCE_IP"`
				InternalIP string `json:"CF_INSTANCE_INTERNAL_IP"`
				Port       string `json:"CF_INSTANCE_PORT"`
				Addr       string `json:"CF_INSTANCE_ADDR"`
				Ports      string `json:"CF_INSTANCE_PORTS"`
			}
			var envValues env
			err := json.Unmarshal([]byte(envOutput), &envValues)
			Expect(err).NotTo(HaveOccurred())
			Expect(envValues.Index).To(Equal("0"))
			Expect(envValues.IP).To(MatchRegexp(`[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+`))
			Expect(envValues.InternalIP).To(MatchRegexp(`[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+`))
			Expect(envValues.Port).To(MatchRegexp(`[0-9]+`))
			Expect(envValues.Addr).To(MatchRegexp(`[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+:[0-9]+`))
			var ports []struct {
				External int `json:"external"`
				Internal int `json:"internal"`
			}
			err = json.Unmarshal([]byte(envValues.Ports), &ports)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(ports)).NotTo(BeZero())
			Expect(ports[0].Internal).NotTo(BeZero())
			Expect(ports[0].External).NotTo(BeZero())
		})

		It("generates an app usage 'started' event", func() {
			Expect(cf.Cf("push",
				appName,
				"--no-start",
				"-b", Config.GetBinaryBuildpackName(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", assets.NewAssets().Catnip,
				"-c", "./catnip",
				"-d", Config.GetAppsDomain()).Wait(Config.DefaultTimeoutDuration()),
			).To(Exit(0))
			app_helpers.SetBackend(appName)

			Expect(cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			found, _ := lastAppUsageEvent(appName, "STARTED")
			Expect(found).To(BeTrue())
		})

		It("generates an app usage 'buildpack_set' event", func() {
			Expect(cf.Cf("push",
				appName,
				"--no-start",
				"-b", Config.GetBinaryBuildpackName(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", assets.NewAssets().Catnip,
				"-c", "./catnip",
				"-d", Config.GetAppsDomain()).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
			app_helpers.SetBackend(appName)

			Expect(cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			found, matchingEvent := lastAppUsageEvent(appName, "BUILDPACK_SET")

			Expect(found).To(BeTrue())
			Expect(matchingEvent.Entity.BuildpackName).To(Equal("binary_buildpack"))
			Expect(matchingEvent.Entity.BuildpackGuid).ToNot(BeZero())
		})
	})

	Describe("stopping", func() {
		BeforeEach(func() {
			Expect(cf.Cf("push",
				appName,
				"--no-start",
				"-b", Config.GetBinaryBuildpackName(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", assets.NewAssets().Catnip,
				"-c", "./catnip",
				"-d", Config.GetAppsDomain()).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
			app_helpers.SetBackend(appName)

			Expect(cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		})

		It("makes the app unreachable", func() {
			Expect(cf.Cf("stop", appName).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}, Config.DefaultTimeoutDuration()).Should(ContainSubstring(expectedNullResponse))
		})

		It("generates an app usage 'stopped' event", func() {
			Expect(cf.Cf("stop", appName).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))

			found, _ := lastAppUsageEvent(appName, "STOPPED")
			Expect(found).To(BeTrue())
		})

		Describe("and then starting", func() {
			It("makes the app reachable again", func() {
				Expect(cf.Cf("stop", appName).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))

				Eventually(func() bool {
					found, _ := lastAppUsageEvent(appName, "STOPPED")
					return found
				}, Config.DefaultTimeoutDuration()).Should(BeTrue())

				Expect(cf.Cf("start", appName).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))

				Eventually(func() string {
					return helpers.CurlAppRoot(Config, appName)
				}, Config.DefaultTimeoutDuration()).Should(ContainSubstring("Catnip?"))
			})
		})
	})

	Describe("updating", func() {
		BeforeEach(func() {
			Expect(cf.Cf("push",
				appName,
				"--no-start",
				"-b", Config.GetBinaryBuildpackName(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", assets.NewAssets().Catnip,
				"-c", "./catnip",
				"-d", Config.GetAppsDomain()).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
			app_helpers.SetBackend(appName)

			Expect(cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		})

		It("is reflected through another push", func() {
			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}, Config.DefaultTimeoutDuration()).Should(ContainSubstring("Catnip?"))

			Expect(cf.Cf("push",
				appName,
				"--no-start",
				"-b", Config.GetRubyBuildpackName(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", assets.NewAssets().HelloWorld,
				"-c", "null",
				"-d", Config.GetAppsDomain()).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
			app_helpers.SetBackend(appName)
			Expect(cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}, Config.DefaultTimeoutDuration()).Should(ContainSubstring("Hello, world!"))
		})
	})

	Describe("deleting", func() {
		var expectedNullResponse string

		BeforeEach(func() {
			appUrl := "https://" + appName + "." + Config.GetAppsDomain()
			nullSession := helpers.CurlSkipSSL(Config.GetSkipSSLValidation(), appUrl).Wait(Config.DefaultTimeoutDuration())
			expectedNullResponse = string(nullSession.Buffer().Contents())

			Expect(cf.Cf("push",
				appName,
				"--no-start",
				"-b", Config.GetBinaryBuildpackName(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", assets.NewAssets().Catnip,
				"-c", "./catnip",
				"-d", Config.GetAppsDomain()).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
			app_helpers.SetBackend(appName)

			Expect(cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		})

		It("removes the application", func() {
			Expect(cf.Cf("delete", appName, "-f", "-r").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))

			app := cf.Cf("apps").Wait(Config.DefaultTimeoutDuration())
			Consistently(app).ShouldNot(Say(appName))
		})

		It("makes the app unreachable", func() {
			Expect(cf.Cf("delete", appName, "-f", "-r").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}, Config.DefaultTimeoutDuration()).Should(ContainSubstring(expectedNullResponse))
		})

		It("generates an app usage 'stopped' event", func() {
			Expect(cf.Cf("delete", appName, "-f", "-r").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))

			found, _ := lastAppUsageEvent(appName, "STOPPED")
			Expect(found).To(BeTrue())
		})
	})
})
