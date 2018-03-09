package credhub

import (
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"encoding/json"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
)

var _ = CredhubDescribe("service bindings", func() {
	var (
		chBrokerAppName string
		chServiceName   string
		instanceName    string
		appStartSession *Session
	)

	BeforeEach(func() {
		if chBrokerAppName == "" {
			chBrokerAppName, chServiceName, instanceName = pushBroker()
		}
	})

	Context("during staging", func() {
		var (
			buildpackName string
			appName       string
		)

		BeforeEach(func() {
			appName, buildpackName = pushBuildpackApp()

			app_helpers.SetBackend(appName)
			appStartSession = bindServiceAndStartApp(chServiceName, instanceName, appName)
		})

		AfterEach(func() {
			app_helpers.AppReport(appName, Config.DefaultTimeoutDuration())

			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				Expect(cf.Cf("delete-buildpack", buildpackName, "-f").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
			})
		})

		NonAssistedCredhubDescribe("", func() {
			It("still contains CredHub references in VCAP_SERVICES", func() {
				Expect(appStartSession).NotTo(Say("pinkyPie"))
				Expect(appStartSession).NotTo(Say("rainbowDash"))
				Expect(appStartSession).To(Say("credhub-ref"))
			})
		})

		AssistedCredhubDescribe("", func() {
			It("has CredHub references in VCAP_SERVICES interpolated", func() {
				Expect(appStartSession).To(Say(`{"password":"rainbowDash","user-name":"pinkyPie"}`))
				Expect(appStartSession).NotTo(Say("credhub-ref"))
			})
		})
	})

	Context("during runtime", func() {
		Describe("service bindings to credhub enabled broker", func() {
			var appName, appURL string
			BeforeEach(func() {
				appName = random_name.CATSRandomName("APP-CH")
				appURL = "https://" + appName + "." + Config.GetAppsDomain()
			})

			AfterEach(func() {
				app_helpers.AppReport(appName, Config.DefaultTimeoutDuration())
				app_helpers.AppReport(chBrokerAppName, Config.DefaultTimeoutDuration())
			})

			NonAssistedCredhubDescribe("", func() {
				BeforeEach(func() {
					createApp := cf.Cf(
						"push", appName,
						"--no-start",
						"-b", Config.GetJavaBuildpackName(),
						"-m", "1024M",
						"-p", assets.NewAssets().CredHubEnabledApp,
						"-d", Config.GetAppsDomain(),
					).Wait(Config.CfPushTimeoutDuration())
					Expect(createApp).To(Exit(0), "failed creating credhub-enabled app")
					appStartSession = bindServiceAndStartApp(chServiceName, instanceName, appName)
				})

				It("leaves the credhub ref in the apps environment", func() {
					appEnv := string(cf.Cf("env", appName).Wait(Config.DefaultTimeoutDuration()).Out.Contents())
					Expect(appEnv).To(ContainSubstring("credentials"), "credential block missing from service")
					Expect(appEnv).To(ContainSubstring("credhub-ref"), "credhub-ref not found")

					curlCmd := helpers.CurlSkipSSL(true, appURL+"/test").Wait(Config.DefaultTimeoutDuration())
					Expect(curlCmd).To(Exit(0))
					var response struct {
						UserName string `json:"user-name"`
						Password string `json:"password"`
					}

					json.Unmarshal(curlCmd.Out.Contents(), &response)
					Expect(response.UserName).To(Equal("pinkyPie"))
					Expect(response.Password).To(Equal("rainbowDash"))
				})
			})

			AssistedCredhubDescribe("", func() {
				BeforeEach(func() {
					createApp := cf.Cf(
						"push", appName,
						"--no-start",
						"-b", Config.GetBinaryBuildpackName(),
						"-m", DEFAULT_MEMORY_LIMIT,
						"-p", assets.NewAssets().Catnip,
						"-c", "./catnip",
						"-d", Config.GetAppsDomain(),
					).Wait(Config.CfPushTimeoutDuration())
					Expect(createApp).To(Exit(0), "failed creating credhub-enabled app")
					appStartSession = bindServiceAndStartApp(chServiceName, instanceName, appName)
				})

				It("interpolates the credentials into VCAP_SERVICES", func() {
					appEnv := string(cf.Cf("env", appName).Wait(Config.DefaultTimeoutDuration()).Out.Contents())
					Expect(appEnv).To(ContainSubstring("credentials"), "credential block missing from service")
					Expect(appEnv).To(ContainSubstring("credhub-ref"), "credhub-ref not found")

					curlCmd := helpers.CurlSkipSSL(true, appURL+"/env/VCAP_SERVICES").Wait(Config.DefaultTimeoutDuration())
					Eventually(curlCmd.Out).Should(Say(`"rainbowDash"`))
					Eventually(curlCmd.Out).Should(Say(`"pinkyPie"`))
				})
			})
		})
	})
})
