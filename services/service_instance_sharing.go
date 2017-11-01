package services_test

import (
	"encoding/json"
	"fmt"
	"strings"

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
			serviceInstanceGuid string
			appName             string
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

				target := cf.Cf("target", "-o", orgName).Wait(Config.DefaultTimeoutDuration())
				Expect(target).To(Exit(0), "failed targeting")

				By("Creating a space that only User A can view")
				userASpaceName := random_name.CATSRandomName("SPACE")
				createSpace := cf.Cf("create-space", userASpaceName, "-o", orgName).Wait(Config.DefaultTimeoutDuration())
				Expect(createSpace).To(Exit(0), "failed to create space")

				target = cf.Cf("target", "-s", userASpaceName).Wait(Config.DefaultTimeoutDuration())
				Expect(target).To(Exit(0), "failed targeting")

				serviceInstanceName = random_name.CATSRandomName("SVIN")

				By("Creating a service instance in User A's space")
				createService := cf.Cf("create-service", broker.Service.Name, broker.SyncPlans[0].Name, serviceInstanceName).Wait(Config.DefaultTimeoutDuration())
				Expect(createService).To(Exit(0))

				By("Sharing the service instance into User B's space")
				serviceInstanceGuid = getGuidFor("service", serviceInstanceName)
				userBSpaceGuid := getGuidFor("space", TestSetup.RegularUserContext().Space)

				shareSpace := cf.Cf("curl",
					fmt.Sprintf("/v3/service_instances/%s/relationships/shared_spaces", serviceInstanceGuid),
					"-X", "POST", "-d", fmt.Sprintf(`{ "data": [ { "guid": "%s" } ] }`, userBSpaceGuid)).Wait(Config.DefaultTimeoutDuration())
				Expect(shareSpace).To(Exit(0))
				Expect(shareSpace).To(Say("data"))
			})
		})

		AfterEach(func() {
			broker.Destroy()

			if appName != "" {
				app_helpers.AppReport(appName, Config.DefaultTimeoutDuration())
				Eventually(cf.Cf("delete", appName, "-f"), Config.DefaultTimeoutDuration()).Should(Exit(0))
			}

			if serviceInstanceName != "" {
				Expect(cf.Cf("delete-service", serviceInstanceName, "-f").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
			}
		})

		It("allows User B to view the shared service", func() {
			workflowhelpers.AsUser(TestSetup.RegularUserContext(), Config.DefaultTimeoutDuration(), func() {
				By("Asserting the User B sees the service instance listed in `cf services`")
				servicesCmd := cf.Cf("services").Wait(Config.DefaultTimeoutDuration())
				Expect(servicesCmd).To(Exit(0))
				Expect(servicesCmd).To(Say(serviceInstanceName))

				By("Asserting the User B sees the service instance in `cf service service-name` output")
				serviceCmd := cf.Cf("service", serviceInstanceName).Wait(Config.DefaultTimeoutDuration())
				Expect(serviceCmd).To(Exit(0))
				Expect(serviceCmd).To(Say("Service instance: " + serviceInstanceName))
				Expect(serviceCmd).To(Say("Service: " + broker.Service.Name))
			})
		})

		It("allows User B to bind an app to the shared service instance", func() {
			workflowhelpers.AsUser(TestSetup.RegularUserContext(), Config.DefaultTimeoutDuration(), func() {
				By("Asserting the User B can bind to the shared service")
				appName = random_name.CATSRandomName("APP")
				Expect(cf.Cf("push",
					appName,
					"-b", Config.GetBinaryBuildpackName(),
					"-m", DEFAULT_MEMORY_LIMIT,
					"-p", assets.NewAssets().Catnip,
					"-c", "./catnip",
					"-d", Config.GetAppsDomain()).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

				bindCmd := cf.Cf("bind-service", appName, serviceInstanceName).Wait(Config.DefaultTimeoutDuration())
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
			var appGuid string

			workflowhelpers.AsUser(TestSetup.RegularUserContext(), Config.DefaultTimeoutDuration(), func() {
				By("Asserting User B can bind to the shared service")
				appName = random_name.CATSRandomName("APP")
				Expect(cf.Cf("push",
					appName,
					"-b", Config.GetBinaryBuildpackName(),
					"-m", DEFAULT_MEMORY_LIMIT,
					"-p", assets.NewAssets().Catnip,
					"-c", "./catnip",
					"-d", Config.GetAppsDomain()).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

				appGuid = getGuidFor("app", appName)

				bindCmd := cf.Cf("bind-service", appName, serviceInstanceName).Wait(Config.DefaultTimeoutDuration())
				Expect(bindCmd).To(Exit(0))
				Expect(bindCmd).To(Say("OK"))

				Expect(cf.Cf("restage", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
			})

			workflowhelpers.AsUser(TestSetup.RegularUserContext(), Config.DefaultTimeoutDuration(), func() {
				By("Asserting User B can unbind from the shared service")
				unbindCmd := cf.Cf("unbind-service", appName, serviceInstanceName).Wait(Config.DefaultTimeoutDuration())
				Expect(unbindCmd).To(Exit(0))
				Expect(unbindCmd).To(Say("OK"))
			})
		})

		It("allows User A to unshare the service regardless of bindings in target space", func() {
			By("Asserting User B can bind to the shared service")
			workflowhelpers.AsUser(TestSetup.RegularUserContext(), Config.DefaultTimeoutDuration(), func() {
				appName = random_name.CATSRandomName("APP")
				Expect(cf.Cf("push",
					appName,
					"-b", Config.GetBinaryBuildpackName(),
					"-m", DEFAULT_MEMORY_LIMIT,
					"-p", assets.NewAssets().Catnip,
					"-c", "./catnip",
					"-d", Config.GetAppsDomain()).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

				bindCmd := cf.Cf("bind-service", appName, serviceInstanceName).Wait(Config.DefaultTimeoutDuration())
				Expect(bindCmd).To(Exit(0))
				Expect(bindCmd).To(Say("OK"))

				Expect(cf.Cf("restage", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
			})

			By("Unsharing the service as User A")
			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				orgName := TestSetup.RegularUserContext().Org
				spaceName := TestSetup.RegularUserContext().Space

				target := cf.Cf("target", "-o", orgName, "-s", spaceName).Wait(Config.DefaultTimeoutDuration())
				Expect(target).To(Exit(0))

				userBSpaceGuid := getGuidFor("space", TestSetup.RegularUserContext().Space)

				unshareSpace := cf.Cf("curl",
					fmt.Sprintf("/v3/service_instances/%s/relationships/shared_spaces/%s", serviceInstanceGuid, userBSpaceGuid),
					"-X", "DELETE").Wait(Config.DefaultTimeoutDuration())

				Expect(unshareSpace).To(Exit(0))
				Expect(unshareSpace).ToNot(Say("errors"))
			})

			By("Asserting the User B can no longer see the service after it has been unshared")
			workflowhelpers.AsUser(TestSetup.RegularUserContext(), Config.DefaultTimeoutDuration(), func() {
				spaceCmd := cf.Cf("services").Wait(Config.DefaultTimeoutDuration())
				Expect(spaceCmd).To(Exit(0))
				Expect(spaceCmd).ToNot(Say(serviceInstanceName))
			})
		})
	})
})

func getGuidFor(resourceType, resourceName string) string {
	session := cf.Cf(resourceType, resourceName, "--guid").Wait(Config.DefaultTimeoutDuration())

	// temporary for: https://github.com/cloudfoundry/cli/issues/1271
	out := string(session.Out.Contents())
	outs := strings.Split(out, "\n")
	return strings.TrimSpace(outs[len(outs)-2])
}
