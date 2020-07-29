package apps

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

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
)

type AppUsageEvent struct {
	App struct {
		Name string
		Guid string
	}
	State struct {
		Current string
		Previous string
	}
	Buildpack struct {
		Guid string
		Name string
	}
	Process struct{
		Guid string
		Name string
	}
}

type AppUsageEvents struct {
	Resources []AppUsageEvent `struct:"resources"`
}

func lastAppUsageEvent(appName string, state string) (bool, AppUsageEvent) {
	var response AppUsageEvents
	workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
		workflowhelpers.ApiRequest("GET", "/v3/app_usage_events?order_by=-created_at&page=1&per_page=150", &response, Config.DefaultTimeoutDuration())
	})

	for _, event := range response.Resources {
		if event.App.Name == appName && event.State.Current == state {
			return true, event
		}
	}

	return false, AppUsageEvent{}
}

func lastAppUsageEventWithParentAppName(parentAppName string, state string) (bool, AppUsageEvent) {
	var response AppUsageEvents
	workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
		workflowhelpers.ApiRequest("GET", "/v3/app_usage_events?order_by=-created_at&page=1&per_page=150", &response, Config.DefaultTimeoutDuration())
	})

	for _, event := range response.Resources {
		if event.App.Name == parentAppName && event.State.Current == state {
			return true, event
		}
	}

	return false, AppUsageEvent{}
}

var _ = Describe("Application Lifecycle", func() {
	var (
		appName              string
		expectedNullResponse string
	)

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")

		appUrl := "https://" + appName + "." + Config.GetAppsDomain()
		nullSession := helpers.CurlSkipSSL(Config.GetSkipSSLValidation(), appUrl).Wait()
		expectedNullResponse = string(nullSession.Buffer().Contents())
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)

		Expect(cf.Cf("delete", appName, "-f", "-r").Wait()).To(Exit(0))
	})

	Describe("pushing", func() {
		It("makes the app reachable via its bound route", func() {
			Expect(cf.Push(appName,
				"-b", Config.GetBinaryBuildpackName(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", assets.NewAssets().Catnip,
				"-c", "./catnip",
			).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}).Should(ContainSubstring("Catnip?"))
		})

		Describe("Context path", func() {
			var app2 string
			var appPath = "/imposter_dora"

			BeforeEach(func() {
				Expect(cf.Push(appName,
					"-b", Config.GetBinaryBuildpackName(),
					"-m", DEFAULT_MEMORY_LIMIT,
					"-p", assets.NewAssets().Catnip,
					"-c", "./catnip",
				).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

				app2 = random_name.CATSRandomName("APP")
				Expect(cf.Push(app2, "-b", Config.GetRubyBuildpackName(), "-m", DEFAULT_MEMORY_LIMIT, "-p", assets.NewAssets().HelloWorld).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
			})

			AfterEach(func() {
				Expect(cf.Cf("delete", app2, "-f", "-r").Wait()).To(Exit(0))
			})

			It("makes another app available via same host and domain, but different path", func() {
				getRoutePath := fmt.Sprintf("/v3/routes?hosts=%s", appName)
				routeBody := cf.Cf("curl", getRoutePath).Wait().Out.Contents()
				var routeJSON struct {
					Resources []struct {
						Relationships struct {
							Space struct {
								Data struct {
									Guid string `json:"guid"`
								} `json:"data"`
							} `json:"space"`
							Domain struct {
								Data struct {
									Guid string `json:"guid"`
								} `json:"data"`
							} `json:"domain"`
						} `json:"relationships"`
					} `json:"resources"`
				}

				Expect(json.Unmarshal([]byte(routeBody), &routeJSON)).To(Succeed())

				Expect(len(routeJSON.Resources)).To(BeNumerically(">=", 1))
				spaceGuid := routeJSON.Resources[0].Relationships.Space.Data.Guid
				domainGuid := routeJSON.Resources[0].Relationships.Domain.Data.Guid
				appGuid := cf.Cf("app", app2, "--guid").Wait().Out.Contents()

				var createRouteBody struct {
					Host string `json:"host"`
					Path string `json:"path"`
					Relationships struct {
						Space struct {
							Data struct {
								Guid string `json:"guid"`
							} `json:"data"`
						} `json:"space"`
						Domain struct {
							Data struct {
								Guid string `json:"guid"`
							} `json:"data"`
						} `json:"domain"`
					} `json:"relationships"`
				}
				createRouteBody.Host = appName
				createRouteBody.Path = appPath
				createRouteBody.Relationships.Space.Data.Guid = spaceGuid
				createRouteBody.Relationships.Domain.Data.Guid = domainGuid

				jsonBody, err := json.Marshal(createRouteBody)
				Expect(err).NotTo(HaveOccurred())

				routePostResponseBody := cf.Cf("curl", "/v3/routes", "-X", "POST", "-d", string(jsonBody)).Wait(Config.CfPushTimeoutDuration()).Out.Contents()
				var routeResponseJSON struct {
					Guid string `json:"guid"`
				}
				json.Unmarshal([]byte(routePostResponseBody), &routeResponseJSON)
				routeGuid := routeResponseJSON.Guid

				type Destination struct {
					App struct {
						Guid string `json:"guid"`
					} `json:"app"`
				}

				var destinationBody struct{
					Destinations []Destination `json:"destinations"`
				}
				destinationBody.Destinations = []Destination{{}}
				destinationBody.Destinations[0].App.Guid = strings.TrimSpace(string(appGuid))

				jsonBody, err = json.Marshal(destinationBody)
				Expect(cf.Cf("curl", "/v3/routes/"+string(routeGuid)+"/destinations", "-X", "POST", "-d", string(jsonBody)).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

				Eventually(func() string {
					return helpers.CurlAppRoot(Config, appName)
				}).Should(ContainSubstring("Catnip?"))

				Eventually(func() string {
					return helpers.CurlApp(Config, appName, appPath)
				}).Should(ContainSubstring("Hello, world!"))
			})
		})

		Context("multiple instances", func() {
			BeforeEach(func() {
				Expect(cf.Push(appName,
					"-b", Config.GetBinaryBuildpackName(),
					"-m", DEFAULT_MEMORY_LIMIT,
					"-p", assets.NewAssets().Catnip,
					"-c", "./catnip",
					"-i", "2",
				).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
			})

			It("is able to start all instances", func() {
				Eventually(func() *Session {
					return cf.Cf("app", appName).Wait()
				}).Should(Say("#0   running"))

				Eventually(func() *Session {
					return cf.Cf("app", appName).Wait()
				}).Should(Say("#1   running"))
			})

			It("is able to retrieve container metrics", func() {
				// #0   running   2015-06-10 02:22:39 PM   0.0%   48.7M of 2G   14M of 1G
				var metrics = regexp.MustCompile(`running.*(?:[\d\.]+)%\s+([\d\.]+)[KMG]? of (?:[\d\.]+)[KMG]\s+([\d\.]+)[KMG]? of (?:[\d\.]+)[KMG]`)
				memdisk := func() (float64, float64) {
					app := cf.Cf("app", appName)
					Expect(app.Wait()).To(Exit(0))

					contents := string(app.Out.Contents())
					arr := metrics.FindStringSubmatch(contents)
					Expect(arr).NotTo(BeNil(), "Regex did not find a match in contents '%s'", contents)
					mem, err := strconv.ParseFloat(arr[1], 64)
					Expect(err).ToNot(HaveOccurred())
					disk, err := strconv.ParseFloat(arr[2], 64)
					Expect(err).ToNot(HaveOccurred())
					return mem, disk
				}
				Eventually(func() float64 { m, _ := memdisk(); return m }, Config.CfPushTimeoutDuration()).Should(BeNumerically(">", 0.0))
				Eventually(func() float64 { _, d := memdisk(); return d }, Config.CfPushTimeoutDuration()).Should(BeNumerically(">", 0.0))
			})

			It("is able to restart an instance", func() {
				idsBefore := app_helpers.ReportedIDs(2, appName)
				Expect(len(idsBefore)).To(Equal(2))
				Expect(cf.Cf("restart-app-instance", appName, "1").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
				Eventually(func() []string {
					return app_helpers.DifferentIDsFrom(idsBefore, appName)
				}, Config.CfPushTimeoutDuration(), 2*time.Second).Should(HaveLen(1))
			})
		})

		It("makes system environment variables available", func() {
			Expect(cf.Push(appName,
				"-b", Config.GetBinaryBuildpackName(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", assets.NewAssets().Catnip,
				"-c", "./catnip",
			).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			var envOutput string
			envOutput = helpers.CurlApp(Config, appName, "/env.json")
			Expect(envOutput).ToNot(BeEmpty())
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
			var ports []struct {
				External *int `json:"external"`
				Internal int  `json:"internal"`
			}
			err = json.Unmarshal([]byte(envValues.Ports), &ports)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(ports)).NotTo(BeZero())
			Expect(ports[0].Internal).NotTo(BeZero())

			if Config.GetRequireProxiedAppTraffic() {
				Expect(ports[0].External).To(BeNil())
				Expect(envValues.Port).To(BeZero())
				Expect(envValues.Addr).To(BeZero())
			} else {
				Expect(ports[0].External).NotTo(BeNil())
				Expect(*ports[0].External).NotTo(BeZero())
				Expect(envValues.Port).To(MatchRegexp(`[0-9]+`))
				Expect(envValues.Addr).To(MatchRegexp(`[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+:[0-9]+`))
			}
		})

		It("generates an app usage 'started' event", func() {
			Expect(cf.Push(appName,
				"-b", Config.GetBinaryBuildpackName(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", assets.NewAssets().Catnip,
				"-c", "./catnip",
			).Wait(Config.CfPushTimeoutDuration()),
			).To(Exit(0))

			found, _ := lastAppUsageEvent(appName, "STARTED")
			Expect(found).To(BeTrue())
		})

		It("generates an app usage 'buildpack_set' event", func() {
			Expect(cf.Push(appName,
				"-b", Config.GetBinaryBuildpackName(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", assets.NewAssets().Catnip,
				"-c", "./catnip",
			).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			found, matchingEvent := lastAppUsageEventWithParentAppName(appName, "BUILDPACK_SET")

			Expect(found).To(BeTrue())
			Expect(matchingEvent.Buildpack.Name).To(Equal("binary_buildpack"))
			Expect(matchingEvent.Buildpack.Guid).ToNot(BeZero())
		})
	})

	Describe("stopping", func() {
		BeforeEach(func() {
			Expect(cf.Push(appName,
				"-b", Config.GetBinaryBuildpackName(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", assets.NewAssets().Catnip,
				"-c", "./catnip",
			).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		})

		It("makes the app unreachable", func() {
			Expect(cf.Cf("stop", appName).Wait()).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}).Should(ContainSubstring(expectedNullResponse))
		})

		It("generates an app usage 'stopped' event", func() {
			Expect(cf.Cf("stop", appName).Wait()).To(Exit(0))

			found, _ := lastAppUsageEvent(appName, "STOPPED")
			Expect(found).To(BeTrue())
		})

		Describe("and then starting", func() {
			It("makes the app reachable again", func() {
				Expect(cf.Cf("stop", appName).Wait()).To(Exit(0))

				Eventually(func() bool {
					found, _ := lastAppUsageEvent(appName, "STOPPED")
					return found
				}).Should(BeTrue())

				Expect(cf.Cf("start", appName).Wait()).To(Exit(0))

				Eventually(func() string {
					return helpers.CurlAppRoot(Config, appName)
				}).Should(ContainSubstring("Catnip?"))
			})
		})
	})

	Describe("updating", func() {
		BeforeEach(func() {
			Expect(cf.Push(appName,
				"-b", Config.GetBinaryBuildpackName(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", assets.NewAssets().Catnip,
				"-c", "./catnip",
			).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		})

		It("is reflected through another push", func() {
			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}).Should(ContainSubstring("Catnip?"))

			Expect(cf.Push(appName,
				"-b", Config.GetRubyBuildpackName(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", assets.NewAssets().HelloWorld,
				"-c", "null",
			).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}).Should(ContainSubstring("Hello, world!"))
		})
	})

	Describe("deleting", func() {
		var expectedNullResponse string

		BeforeEach(func() {
			appUrl := "https://" + appName + "." + Config.GetAppsDomain()
			nullSession := helpers.CurlSkipSSL(Config.GetSkipSSLValidation(), appUrl).Wait()
			expectedNullResponse = string(nullSession.Buffer().Contents())

			Expect(cf.Push(appName,
				"-b", Config.GetBinaryBuildpackName(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", assets.NewAssets().Catnip,
				"-c", "./catnip",
			).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		})

		It("removes the application", func() {
			Expect(cf.Cf("delete", appName, "-f", "-r").Wait()).To(Exit(0))

			app := cf.Cf("apps").Wait()
			Consistently(app).ShouldNot(Say(appName))
		})

		It("makes the app unreachable", func() {
			Expect(cf.Cf("delete", appName, "-f", "-r").Wait()).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}).Should(ContainSubstring(expectedNullResponse))
		})

		It("generates an app usage 'stopped' event", func() {
			Expect(cf.Cf("delete", appName, "-f", "-r").Wait()).To(Exit(0))

			found, _ := lastAppUsageEvent(appName, "STOPPED")
			Expect(found).To(BeTrue())
		})
	})
})
