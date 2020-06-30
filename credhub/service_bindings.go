package credhub

import (
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"encoding/json"

	"strings"

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
		TestSetup.RegularUserContext().TargetSpace()
		cf.Cf("target", "-o", TestSetup.RegularUserContext().Org)

		chBrokerAppName = random_name.CATSRandomName("BRKR-CH")

		Expect(cf.Cf(
			"push", chBrokerAppName,
			"-b", Config.GetGoBuildpackName(),
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", assets.NewAssets().CredHubServiceBroker,
			"-f", assets.NewAssets().CredHubServiceBroker+"/manifest.yml",
			"-d", Config.GetAppsDomain(),
		).Wait(Config.CfPushTimeoutDuration())).To(Exit(0), "failed pushing credhub-enabled service broker")

		existingEnvVar := string(cf.Cf("running-environment-variable-group").Wait().Out.Contents())

		if !strings.Contains(existingEnvVar, "CREDHUB_API") {
			Expect(cf.Cf(
				"set-env", chBrokerAppName,
				"CREDHUB_API", Config.GetCredHubLocation(),
			).Wait()).To(Exit(0), "failed setting CREDHUB_API env var on credhub-enabled service broker")
		}

		chServiceName = random_name.CATSRandomName("SERVICE-NAME")
		Expect(cf.Cf(
			"set-env", chBrokerAppName,
			"SERVICE_NAME", chServiceName,
		).Wait()).To(Exit(0), "failed setting SERVICE_NAME env var on credhub-enabled service broker")

		Expect(cf.Cf(
			"set-env", chBrokerAppName,
			"CREDHUB_CLIENT", Config.GetCredHubBrokerClientCredential(),
		).Wait()).To(Exit(0), "failed setting CREDHUB_CLIENT env var on credhub-enabled service broker")

		Expect(cf.CfRedact(
			Config.GetCredHubBrokerClientSecret(), "set-env", chBrokerAppName,
			"CREDHUB_SECRET", Config.GetCredHubBrokerClientSecret(),
		).Wait()).To(Exit(0), "failed setting CREDHUB_SECRET env var on credhub-enabled service broker")

		Expect(cf.Cf(
			"restart", chBrokerAppName,
		).Wait(Config.CfPushTimeoutDuration())).To(Exit(0), "failed restarting credhub-enabled service broker")

		workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
			serviceUrl := "https://" + chBrokerAppName + "." + Config.GetAppsDomain()
			createServiceBroker := cf.Cf("create-service-broker", chBrokerAppName, Config.GetAdminUser(), Config.GetAdminPassword(), serviceUrl).Wait()
			Expect(createServiceBroker).To(Exit(0), "failed creating credhub-enabled service broker")

			enableAccess := cf.Cf("enable-service-access", chServiceName, "-o", TestSetup.RegularUserContext().Org).Wait()
			Expect(enableAccess).To(Exit(0), "failed to enable service access for credhub-enabled broker")

			TestSetup.RegularUserContext().TargetSpace()
			instanceName = random_name.CATSRandomName("SVIN-CH")
			createService := cf.Cf("create-service", chServiceName, "credhub-read-plan", instanceName).Wait()
			Expect(createService).To(Exit(0), "failed creating credhub enabled service")
		})
	})

	AfterEach(func() {
		workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
			TestSetup.RegularUserContext().TargetSpace()

			Expect(cf.Cf("purge-service-instance", instanceName, "-f").Wait()).To(Exit(0))
			Expect(cf.Cf("delete-service-broker", chBrokerAppName, "-f").Wait()).To(Exit(0))
		})
	})

	bindServiceAndStartApp := func(appName string) {
		Expect(chServiceName).ToNot(Equal(""))
		setServiceName := cf.Cf("set-env", appName, "SERVICE_NAME", chServiceName).Wait()
		Expect(setServiceName).To(Exit(0), "failed setting SERVICE_NAME env var on app")

		existingEnvVar := string(cf.Cf("running-environment-variable-group").Wait().Out.Contents())

		if !strings.Contains(existingEnvVar, "CREDHUB_API") {
			Expect(cf.Cf(
				"set-env", appName,
				"CREDHUB_API", Config.GetCredHubLocation(),
			).Wait()).To(Exit(0), "failed setting CREDHUB_API env var on app")
		}

		workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
			TestSetup.RegularUserContext().TargetSpace()

			bindService := cf.Cf("bind-service", appName, instanceName).Wait()
			Expect(bindService).To(Exit(0), "failed binding app to service")
		})
		appStartSession = cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())
		Expect(appStartSession).To(Exit(0))
	}

	Context("during runtime", func() {
		Describe("service bindings to credhub enabled broker", func() {
			var appName, appURL string
			BeforeEach(func() {
				appName = random_name.CATSRandomName("APP-CH")
				appURL = "https://" + appName + "." + Config.GetAppsDomain()
			})

			AfterEach(func() {
				app_helpers.AppReport(appName)
				app_helpers.AppReport(chBrokerAppName)

				workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
					TestSetup.RegularUserContext().TargetSpace()
					unbindService := cf.Cf("unbind-service", appName, instanceName).Wait()
					Expect(unbindService).To(Exit(0), "failed unbinding app and service")

					Expect(cf.Cf("delete", appName, "-f", "-r").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
				})
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
					bindServiceAndStartApp(appName)
				})

				It("the broker returns credhub-ref in the credentials block", func() {
					appEnv := string(cf.Cf("env", appName).Wait().Out.Contents())
					Expect(appEnv).To(ContainSubstring("credentials"), "credential block missing from service")
					Expect(appEnv).To(ContainSubstring("credhub-ref"), "credhub-ref not found")
				})

				It("the bound app retrieves the credentials for the ref from CredHub", func() {
					curlCmd := helpers.CurlSkipSSL(true, appURL+"/test").Wait()
					Expect(curlCmd).To(Exit(0))

					bytes := curlCmd.Out.Contents()
					var response struct {
						UserName string `json:"user-name"`
						Password string `json:"password"`
					}

					json.Unmarshal(bytes, &response)
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
					bindServiceAndStartApp(appName)
				})

				It("the broker returns credhub-ref in the credentials block", func() {
					appEnv := string(cf.Cf("env", appName).Wait().Out.Contents())
					Expect(appEnv).To(ContainSubstring("credentials"), "credential block missing from service")
					Expect(appEnv).To(ContainSubstring("credhub-ref"), "credhub-ref not found")
				})

				It("the bound app gets CredHub refs in VCAP_SERVICES interpolated", func() {
					curlCmd := helpers.CurlSkipSSL(true, appURL+"/env/VCAP_SERVICES").Wait()
					Expect(curlCmd).To(Exit(0))

					bytes := curlCmd.Out.Contents()
					Expect(string(bytes)).To(ContainSubstring(`"rainbowDash"`))
					Expect(string(bytes)).To(ContainSubstring(`"pinkyPie"`))
				})
			})
		})
	})
})
