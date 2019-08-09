package docker

import (
	"fmt"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"strings"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/skip_messages"
)

var _ = DockerDescribe("Docker App Lifecycle CredHub Integration", func() {
	Context("when CredHub is configured", func() {
		var chBrokerName, chServiceName, instanceName string

		JustBeforeEach(func() {
			TestSetup.RegularUserContext().TargetSpace()
			cf.Cf("target", "-o", TestSetup.RegularUserContext().Org)

			chBrokerName = random_name.CATSRandomName("BRKR-CH")

			pushBroker := cf.Push(chBrokerName, "-b", Config.GetGoBuildpackName(), "-m", DEFAULT_MEMORY_LIMIT, "-p", assets.NewAssets().CredHubServiceBroker, "-f", assets.NewAssets().CredHubServiceBroker+"/manifest.yml", "-d", Config.GetAppsDomain()).Wait(Config.CfPushTimeoutDuration())
			Expect(pushBroker).To(Exit(0), "failed pushing credhub-enabled service broker")

			existingEnvVar := string(cf.Cf("running-environment-variable-group").Wait().Out.Contents())

			if !strings.Contains(existingEnvVar, "CREDHUB_API") {
				Expect(cf.Cf(
					"set-env", chBrokerName,
					"CREDHUB_API", Config.GetCredHubLocation(),
				).Wait()).To(Exit(0), "failed setting CREDHUB_API env var on credhub-enabled service broker")
			}

			Expect(cf.Cf(
				"set-env", chBrokerName,
				"CREDHUB_CLIENT", Config.GetCredHubBrokerClientCredential(),
			).Wait()).To(Exit(0), "failed setting CREDHUB_CLIENT env var on credhub-enabled service broker")

			Expect(cf.CfRedact(
				Config.GetCredHubBrokerClientSecret(), "set-env", chBrokerName,
				"CREDHUB_SECRET", Config.GetCredHubBrokerClientSecret(),
			).Wait()).To(Exit(0), "failed setting CREDHUB_SECRET env var on credhub-enabled service broker")

			chServiceName = random_name.CATSRandomName("SERVICE-NAME")
			setServiceName := cf.Cf("set-env", chBrokerName, "SERVICE_NAME", chServiceName).Wait()
			Expect(setServiceName).To(Exit(0), "failed setting SERVICE_NAME env var on credhub-enabled service broker")

			restartBroker := cf.Cf("restart", chBrokerName).Wait(Config.CfPushTimeoutDuration())
			Expect(restartBroker).To(Exit(0), "failed restarting credhub-enabled service broker")

			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				serviceUrl := "https://" + chBrokerName + "." + Config.GetAppsDomain()
				createServiceBroker := cf.Cf("create-service-broker", chBrokerName, Config.GetAdminUser(), Config.GetAdminPassword(), serviceUrl).Wait()
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

				Expect(cf.Cf("delete-service", instanceName, "-f").Wait()).To(Exit(0))
				Expect(cf.Cf("purge-service-offering", chServiceName, "-f").Wait()).To(Exit(0))
				Expect(cf.Cf("delete-service-broker", chBrokerName, "-f").Wait()).To(Exit(0))
			})
		})

		Describe("service bindings", func() {
			var appName, dockerImage string

			JustBeforeEach(func() {
				appName = random_name.CATSRandomName("APP-CH")
				Eventually(cf.Cf(
					"push", appName,
					"--no-start",
					// app is defined by cloudfoundry-incubator/diego-dockerfiles
					"-o", dockerImage,
					"-m", DEFAULT_MEMORY_LIMIT,
					"-d", Config.GetAppsDomain(),
					"-i", "1",
					"-c", fmt.Sprintf("/myapp/dockerapp -name=%s", appName)),
				).Should(Exit(0))

				workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
					TestSetup.RegularUserContext().TargetSpace()

					bindService := cf.Cf("bind-service", appName, instanceName).Wait()
					Expect(bindService).To(Exit(0), "failed binding app to service")
					Expect(cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
				})
			})

			AfterEach(func() {
				app_helpers.AppReport(appName)

				workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
					TestSetup.RegularUserContext().TargetSpace()
					unbindService := cf.Cf("unbind-service", appName, instanceName).Wait()
					Expect(unbindService).To(Exit(0), "failed unbinding app and service")

					Expect(cf.Cf("delete", appName, "-f", "-r").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
				})
			})

			Context("in assisted mode", func() {
				BeforeEach(func() {
					if !Config.GetIncludeCredhubAssisted() {
						Skip(skip_messages.SkipAssistedCredhubMessage)
					}

					dockerImage = Config.GetPublicDockerAppImage()
				})

				It("the app should see the service creds", func() {
					env := helpers.CurlApp(Config, appName, "/env")
					Expect(env).NotTo(ContainSubstring("credhub-ref"), "credhub-ref not found")
					Expect(env).To(ContainSubstring("pinkyPie"))
					Expect(env).To(ContainSubstring("rainbowDash"))
				})
			})

			Context("in non-assisted mode", func() {
				BeforeEach(func() {
					if !Config.GetIncludeCredhubNonAssisted() {
						Skip(skip_messages.SkipNonAssistedCredhubMessage)
					}

					// TODO: use the credhub enabled app docker image and interpolate the vcap_services manually
					dockerImage = Config.GetPublicDockerAppImage()
				})

				It("the app should not automatically see the service creds", func() {
					env := helpers.CurlApp(Config, appName, "/env")
					Expect(env).To(ContainSubstring("credhub-ref"), "credhub-ref not found")
					Expect(env).NotTo(ContainSubstring("pinkyPie"))
					Expect(env).NotTo(ContainSubstring("rainbowDash"))
				})
			})
		})
	})
})
