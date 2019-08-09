package services_test

import (
	"encoding/json"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/services"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = ServiceInstanceSharingDescribe("Service Instance Sharing", func() {
	Context("when User A shares a service instance into User B's space", func() {
		// Note: user A is admin and user B is regular user
		var (
			broker              services.ServiceBroker
			serviceInstanceName string
			appName             string
			userASpaceName      string
		)

		BeforeEach(func() {
			broker = services.NewServiceBroker(
				random_name.CATSRandomName("BRKR"),
				assets.NewAssets().ServiceBroker,
				TestSetup,
			)

			broker.Push(Config)
			broker.Configure()
			broker.Create()
			broker.PublicizePlans()

			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				orgName := TestSetup.RegularUserContext().Org

				target := cf.Cf("target", "-o", orgName).Wait()
				Expect(target).To(Exit(0), "failed targeting")

				By("Creating a space that only User A can view")
				userASpaceName = random_name.CATSRandomName("SPACE")
				createSpace := cf.Cf("create-space", userASpaceName, "-o", orgName).Wait()
				Expect(createSpace).To(Exit(0), "failed to create space")

				target = cf.Cf("target", "-s", userASpaceName).Wait()
				Expect(target).To(Exit(0), "failed targeting")

				serviceInstanceName = random_name.CATSRandomName("SVIN")

				By("Creating a service instance in User A's space")
				createService := cf.Cf("create-service", broker.Service.Name, broker.SyncPlans[0].Name, serviceInstanceName).Wait()
				Expect(createService).To(Exit(0))

				By("Sharing the service instance into User B's space")
				userBSpaceName := TestSetup.RegularUserContext().TestSpace.SpaceName()

				shareSpace := cf.Cf("share-service", serviceInstanceName, "-s", userBSpaceName).Wait()

				Expect(shareSpace).To(Exit(0), "failed to share")
				Expect(shareSpace).To(Say("OK"))
			})
		})

		AfterEach(func() {
			broker.Destroy()

			if appName != "" {
				app_helpers.AppReport(appName)
				Eventually(cf.Cf("delete", appName, "-f")).Should(Exit(0))
			}

			if serviceInstanceName != "" {
				Expect(cf.Cf("delete-service", serviceInstanceName, "-f").Wait()).To(Exit(0))
			}
		})

		It("allows User B to view the shared service", func() {
			workflowhelpers.AsUser(TestSetup.RegularUserContext(), Config.DefaultTimeoutDuration(), func() {
				By("Asserting the User B sees the service instance listed in `cf services`")
				servicesCmd := cf.Cf("services").Wait()
				Expect(servicesCmd).To(Exit(0))
				Expect(servicesCmd).To(Say(serviceInstanceName))

				By("Asserting the User B sees the service instance in `cf service service-name` output")
				serviceCmd := cf.Cf("service", serviceInstanceName).Wait()
				Expect(serviceCmd).To(Exit(0))
				Expect(serviceCmd).To(Say(serviceInstanceName))
				Expect(serviceCmd).To(Say(broker.Service.Name))
				Expect(serviceCmd).To(Say("create succeeded"))
			})
		})

		It("allows User A to view share information about the service instance", func() {
			By("Asserting User B can bind to the shared service")
			workflowhelpers.AsUser(TestSetup.RegularUserContext(), Config.DefaultTimeoutDuration(), func() {
				appName = random_name.CATSRandomName("APP")
				Expect(cf.Push(appName,
					"-b", Config.GetBinaryBuildpackName(),
					"-m", DEFAULT_MEMORY_LIMIT,
					"-p", assets.NewAssets().Catnip,
					"-c", "./catnip",
					"-d", Config.GetAppsDomain()).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

				bindCmd := cf.Cf("bind-service", appName, serviceInstanceName).Wait()
				Expect(bindCmd).To(Exit(0))
				Expect(bindCmd).To(Say("OK"))
			})

			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				By("Asserting the User A sees the share information for the shared service")
				orgName := TestSetup.RegularUserContext().Org

				target := cf.Cf("target", "-o", orgName, "-s", userASpaceName).Wait()
				Expect(target).To(Exit(0))

				sharedToCmd := cf.Cf("service", serviceInstanceName).Wait()
				Expect(sharedToCmd).To(Exit(0))
				Expect(sharedToCmd).To(Say("shared with spaces"))
			})
		})

		It("allows User B to bind an app to the shared service instance", func() {
			workflowhelpers.AsUser(TestSetup.RegularUserContext(), Config.DefaultTimeoutDuration(), func() {
				By("Asserting the User B can bind to the shared service")
				appName = random_name.CATSRandomName("APP")
				Expect(cf.Push(appName,
					"-b", Config.GetBinaryBuildpackName(),
					"-m", DEFAULT_MEMORY_LIMIT,
					"-p", assets.NewAssets().Catnip,
					"-c", "./catnip",
					"-d", Config.GetAppsDomain()).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

				bindCmd := cf.Cf("bind-service", appName, serviceInstanceName).Wait()
				Expect(bindCmd).To(Exit(0))
				Expect(bindCmd).To(Say("OK"))

				Expect(cf.Cf("restage", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
				envJSON := helpers.CurlApp(Config, appName, "/env.json")
				var envVars map[string]string
				json.Unmarshal([]byte(envJSON), &envVars)

				Expect(envVars["VCAP_SERVICES"]).To(ContainSubstring("credentials"))
			})
		})

		It("allows User B to unbind an app from the shared service instance", func() {
			workflowhelpers.AsUser(TestSetup.RegularUserContext(), Config.DefaultTimeoutDuration(), func() {
				By("Asserting User B can bind to the shared service")
				appName = random_name.CATSRandomName("APP")
				Expect(cf.Push(appName,
					"-b", Config.GetBinaryBuildpackName(),
					"-m", DEFAULT_MEMORY_LIMIT,
					"-p", assets.NewAssets().Catnip,
					"-c", "./catnip",
					"-d", Config.GetAppsDomain()).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

				bindCmd := cf.Cf("bind-service", appName, serviceInstanceName).Wait()
				Expect(bindCmd).To(Exit(0))
				Expect(bindCmd).To(Say("OK"))

				Expect(cf.Cf("restage", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
			})

			workflowhelpers.AsUser(TestSetup.RegularUserContext(), Config.DefaultTimeoutDuration(), func() {
				By("Asserting User B can unbind from the shared service")
				unbindCmd := cf.Cf("unbind-service", appName, serviceInstanceName).Wait()
				Expect(unbindCmd).To(Exit(0))
				Expect(unbindCmd).To(Say("OK"))
			})
		})

		It("allows User A to unshare the service regardless of bindings in target space", func() {
			By("Asserting User B can bind to the shared service")
			workflowhelpers.AsUser(TestSetup.RegularUserContext(), Config.DefaultTimeoutDuration(), func() {
				appName = random_name.CATSRandomName("APP")
				Expect(cf.Push(appName,
					"-b", Config.GetBinaryBuildpackName(),
					"-m", DEFAULT_MEMORY_LIMIT,
					"-p", assets.NewAssets().Catnip,
					"-c", "./catnip",
					"-d", Config.GetAppsDomain()).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

				bindCmd := cf.Cf("bind-service", appName, serviceInstanceName).Wait()
				Expect(bindCmd).To(Exit(0))
				Expect(bindCmd).To(Say("OK"))

				Expect(cf.Cf("restage", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
			})

			By("Unsharing the service as User A")
			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				orgName := TestSetup.RegularUserContext().Org

				target := cf.Cf("target", "-o", orgName, "-s", userASpaceName).Wait()
				Expect(target).To(Exit(0))

				userBSpaceName := TestSetup.RegularUserContext().TestSpace.SpaceName()

				unshareSpace := cf.Cf("unshare-service", serviceInstanceName, "-s", userBSpaceName, "-f").Wait()
				Expect(unshareSpace).To(Exit(0))
				Expect(unshareSpace).ToNot(Say("errors"))
			})

			By("Asserting the User B can no longer see the service after it has been unshared")
			workflowhelpers.AsUser(TestSetup.RegularUserContext(), Config.DefaultTimeoutDuration(), func() {
				spaceCmd := cf.Cf("services").Wait()
				Expect(spaceCmd).To(Exit(0))
				Expect(spaceCmd).ToNot(Say(serviceInstanceName))
			})
		})
	})
})
