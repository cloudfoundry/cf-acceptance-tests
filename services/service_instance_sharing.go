package services_test

import (
	"fmt"
	"strings"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/services"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = ServicesDescribe("Service Instance Sharing", func() {
	Context("when User A shares a service instance into User B's space", func() {
		var (
			broker              services.ServiceBroker
			serviceInstanceName string
			orgName             string
		)
		// Note: user A is admin and user B is regular user
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

			var userBSpaceGuid string

			workflowhelpers.AsUser(TestSetup.RegularUserContext(), Config.DefaultTimeoutDuration(), func() {
				spaceCmd := cf.Cf("space", TestSetup.RegularUserContext().Space, "--guid").Wait(Config.DefaultTimeoutDuration())
				spaceGuid := string(spaceCmd.Out.Contents())
				userBSpaceGuid = strings.Trim(spaceGuid, "\n")
			})

			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				orgName = random_name.CATSRandomName("ORG")
				createOrg := cf.Cf("create-org", orgName).Wait(Config.DefaultTimeoutDuration())
				Expect(createOrg).To(Exit(0), "failed to create org")

				spaceName := random_name.CATSRandomName("SPACE")
				createSpace := cf.Cf("create-space", spaceName, "-o", orgName).Wait(Config.DefaultTimeoutDuration())
				Expect(createSpace).To(Exit(0), "failed to create space")

				target := cf.Cf("target", "-o", orgName, "-s", spaceName).Wait(Config.DefaultTimeoutDuration())
				Expect(target).To(Exit(0), "failed targeting")

				serviceInstanceName = random_name.CATSRandomName("SVIN")

				createService := cf.Cf("create-service", broker.Service.Name, broker.SyncPlans[0].Name, serviceInstanceName).Wait(Config.DefaultTimeoutDuration())
				Eventually(createService).Should(Exit(0))

				serviceCmd := cf.Cf("service", serviceInstanceName, "--guid").Wait(Config.DefaultTimeoutDuration())
				instanceGuid := string(serviceCmd.Out.Contents())
				instanceGuid = strings.Trim(instanceGuid, "\n")

				shareSpace := cf.Cf("curl",
					fmt.Sprintf("/v3/service_instances/%s/relationships/shared_spaces", instanceGuid),
					"-X", "POST", "-d", fmt.Sprintf(`{ "data": [ { "guid": "%s" } ] }`, userBSpaceGuid))
				Eventually(shareSpace).Should(Exit(0))
				Eventually(shareSpace).Should(Say("data"))
			})
		})

		AfterEach(func() {
			broker.Destroy()
			if serviceInstanceName != "" {
				Expect(cf.Cf("delete-service", serviceInstanceName, "-f").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
			}
			if orgName != "" {
				workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
					Expect(cf.Cf("delete-org", orgName, "-f").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
				})
			}
		})

		It("allows User B to view the shared service instance", func() {
			workflowhelpers.AsUser(TestSetup.RegularUserContext(), Config.DefaultTimeoutDuration(), func() {
				spaceCmd := cf.Cf("services").Wait(Config.DefaultTimeoutDuration())
				Expect(spaceCmd).To(Exit(0))
				Expect(spaceCmd).To(Say(serviceInstanceName))
			})
		})
	})
})
