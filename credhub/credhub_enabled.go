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
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/skip_messages"
)

var _ = CredHubDescribe("CredHub Integration", func() {
	BeforeEach(func() {
		if Config.GetBackend() != "diego" {
			Skip(skip_messages.SkipDiegoMessage)
		}
	})

	Context("when CredHub is configured", func() {
		var chBrokerName, chServiceName, instanceName string

		BeforeEach(func() {
			TestSetup.RegularUserContext().TargetSpace()
			cf.Cf("target", "-o", TestSetup.RegularUserContext().Org)
			Expect(string(cf.Cf("running-environment-variable-group").Wait(Config.DefaultTimeoutDuration()).Out.Contents())).To(ContainSubstring("CREDHUB_API"), "CredHub API environment not set")

			chBrokerName = random_name.CATSRandomName("BRKR-CH")

			pushBroker := cf.Cf("push", chBrokerName, "-b", Config.GetGoBuildpackName(), "-m", DEFAULT_MEMORY_LIMIT, "-p", assets.NewAssets().CredHubServiceBroker, "-f", assets.NewAssets().CredHubServiceBroker+"/manifest.yml", "-d", Config.GetAppsDomain()).Wait(Config.DefaultTimeoutDuration())
			Expect(pushBroker).To(Exit(0), "failed pushing credhub-enabled service broker")

			chServiceName = random_name.CATSRandomName("SERVICE-NAME")
			setServiceName := cf.Cf("set-env", chBrokerName, "SERVICE_NAME", chServiceName).Wait(Config.DefaultTimeoutDuration())
			Expect(setServiceName).To(Exit(0), "failed setting SERVICE_NAME env var on credhub-enabled service broker")

			restartBroker := cf.Cf("restart", chBrokerName).Wait(Config.CfPushTimeoutDuration())
			Expect(restartBroker).To(Exit(0), "failed restarting credhub-enabled service broker")

			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				serviceUrl := "https://" + chBrokerName + "." + Config.GetAppsDomain()
				createServiceBroker := cf.Cf("create-service-broker", chBrokerName, Config.GetAdminUser(), Config.GetAdminPassword(), serviceUrl).Wait(Config.DefaultTimeoutDuration())
				Expect(createServiceBroker).To(Exit(0), "failed creating credhub-enabled service broker")

				enableAccess := cf.Cf("enable-service-access", chServiceName, "-o", TestSetup.RegularUserContext().Org).Wait(Config.DefaultTimeoutDuration())
				Expect(enableAccess).To(Exit(0), "failed to enable service access for credhub-enabled broker")

				TestSetup.RegularUserContext().TargetSpace()
				instanceName = random_name.CATSRandomName("SVIN-CH")
				createService := cf.Cf("create-service", chServiceName, "credhub-read-plan", instanceName).Wait(Config.DefaultTimeoutDuration())
				Expect(createService).To(Exit(0), "failed creating credhub enabled service")
			})
		})

		AfterEach(func() {
			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				TestSetup.RegularUserContext().TargetSpace()

				Expect(cf.Cf("delete-service", instanceName, "-f").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
				Expect(cf.Cf("purge-service-offering", chServiceName).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
				Expect(cf.Cf("delete-service-broker", chBrokerName, "-f").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
			})
		})

		Describe("service bindings", func() {
			var appName, appURL string

			BeforeEach(func() {
				appName = random_name.CATSRandomName("APP-CH")
				appURL = "https://" + appName + "." + Config.GetAppsDomain()
				createApp := cf.Cf("push", appName, "--no-start", "-b", Config.GetJavaBuildpackName(), "-m", "1024M", "-p", assets.NewAssets().CredHubEnabledApp, "-d", Config.GetAppsDomain()).Wait(Config.DefaultTimeoutDuration())
				Expect(createApp).To(Exit(0), "failed creating credhub-enabled app")
				app_helpers.SetBackend(appName)

				Expect(chServiceName).ToNot(Equal(""))
				setServiceName := cf.Cf("set-env", appName, "SERVICE_NAME", chServiceName).Wait(Config.DefaultTimeoutDuration())
				Expect(setServiceName).To(Exit(0), "failed setting SERVICE_NAME env var on app")

				workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
					TestSetup.RegularUserContext().TargetSpace()

					bindService := cf.Cf("bind-service", appName, instanceName).Wait(Config.DefaultTimeoutDuration())
					Expect(bindService).To(Exit(0), "failed binding app to service")
					Expect(cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
				})
			})

			AfterEach(func() {
				app_helpers.AppReport(appName, Config.DefaultTimeoutDuration())

				workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
					TestSetup.RegularUserContext().TargetSpace()
					unbindService := cf.Cf("unbind-service", appName, instanceName).Wait(Config.DefaultTimeoutDuration())
					Expect(unbindService).To(Exit(0), "failed unbinding app and service")

					Expect(cf.Cf("delete", appName, "-f", "-r").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
				})
			})

			Context("when CredHub enabled broker is bound to application", func() {
				It("the broker returns credhub-ref in the credentials block", func() {
					restageApp := cf.Cf("restage", appName).Wait(Config.CfPushTimeoutDuration())
					Expect(restageApp).To(Exit(0), "failed restaging app")

					appEnv := string(cf.Cf("env", appName).Wait(Config.DefaultTimeoutDuration()).Out.Contents())

					Expect(appEnv).To(ContainSubstring("credentials"), "credential block missing from service")
					Expect(appEnv).To(ContainSubstring("credhub-ref"), "credhub-ref not found")
				})

				It("the bound app retrieves the credentials for the ref from CredHub", func() {
					curlCmd := helpers.CurlSkipSSL(true, appURL+"/test").Wait(Config.DefaultTimeoutDuration())
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
		})

		Describe("service keys", func() {
			var serviceKeyName string

			AfterEach(func() {
				workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
					TestSetup.RegularUserContext().TargetSpace()

					Expect(cf.Cf("delete-service-key", instanceName, serviceKeyName, "-f").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
				})
			})

			Context("when a service key for a service instance is requested from a CredHub-enabled broker", func() {
				It("Cloud Controller retrieves the value from CredHub for the service key", func() {
					serviceKeyName = random_name.CATSRandomName("SVKEY-CH")
					createKey := cf.Cf("create-service-key", instanceName, serviceKeyName).Wait(Config.DefaultTimeoutDuration())
					Expect(createKey).To(Exit(0), "failed to create key")

					keyInfo := cf.Cf("service-key", instanceName, serviceKeyName).Wait(Config.DefaultTimeoutDuration())
					Expect(keyInfo).To(Exit(0), "failed key info")

					Expect(keyInfo).To(Say(`"password": "rainbowDash"`))
					Expect(keyInfo).To(Say(`"user-name": "pinkyPie"`))
				})
			})
		})
	})
})
