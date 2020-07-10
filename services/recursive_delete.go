package services_test

import (
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/services"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = ServicesDescribe("Recursive Delete", func() {
	var broker ServiceBroker
	var orgName string
	var quotaName, spaceName, appName, instanceName string

	BeforeEach(func() {
		broker = NewServiceBroker(
			random_name.CATSRandomName("BRKR"),
			assets.NewAssets().ServiceBroker,
			TestSetup,
		)
		broker.Push(Config)
		broker.Configure()
		broker.Create()
		broker.PublicizePlans()

		orgName = random_name.CATSRandomName("ORG")
		quotaName = random_name.CATSRandomName("QUOTA")
		spaceName = random_name.CATSRandomName("SPACE")
		appName = random_name.CATSRandomName("APP")
		instanceName = random_name.CATSRandomName("SVIN")

		workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
			createQuota := cf.Cf("create-quota", quotaName, "-m", "10G", "-r", "1000", "-s", "5").Wait(TestSetup.ShortTimeout())
			Expect(createQuota).To(Exit(0))

			createOrg := cf.Cf("create-org", orgName).Wait()
			Expect(createOrg).To(Exit(0), "failed to create org")

			setQuota := cf.Cf("set-quota", orgName, quotaName).Wait(TestSetup.ShortTimeout())
			Expect(setQuota).To(Exit(0))

			createSpace := cf.Cf("create-space", spaceName, "-o", orgName).Wait()
			Expect(createSpace).To(Exit(0), "failed to create space")

			target := cf.Cf("target", "-o", orgName, "-s", spaceName).Wait()
			Expect(target).To(Exit(0), "failed targeting")

			createApp := cf.Cf("push",
				appName,
				"-b", Config.GetBinaryBuildpackName(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", assets.NewAssets().Catnip,
				"-c", "./catnip",
				"-d", Config.GetAppsDomain()).Wait(Config.CfPushTimeoutDuration())
			Expect(createApp).To(Exit(0), "failed creating app")

			createService := cf.Cf("create-service", broker.Service.Name, broker.SyncPlans[0].Name, instanceName).Wait()
			Expect(createService).To(Exit(0), "failed creating service")
		})
	})

	AfterEach(func() {
		app_helpers.AppReport(broker.Name)

		broker.Destroy()
		workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
			targetOrg := cf.Cf("target", "-o", orgName).Wait()
			if targetOrg.ExitCode() == 0 {
				targetSpace := cf.Cf("target", "-s", spaceName).Wait()
				if targetSpace.ExitCode() == 0 {
					Expect(cf.Cf("delete", appName, "-f").Wait()).To(Exit(0))
					Expect(cf.Cf("delete-service", instanceName, "-f").Wait()).To(Exit(0))
					Expect(cf.Cf("delete-space", spaceName, "-f").Wait()).To(Exit(0))
				}
				Expect(cf.Cf("delete-org", orgName, "-f").Wait()).To(Exit(0))
				Expect(cf.Cf("delete-quota", "-f", quotaName).Wait(TestSetup.ShortTimeout())).To(Exit(0))
			}
		})
	})

	It("deletes all apps and services in all spaces in an org", func() {
		workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
			deleteOrg := cf.Cf("delete-org", orgName, "-f").Wait()
			Expect(deleteOrg).To(Exit(0), "failed deleting org")
		})
		getOrg := cf.Cf("org", orgName).Wait()
		Expect(getOrg).To(Exit(1), "org still exists")
	})
})
