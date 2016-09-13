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
	var quotaName string

	BeforeEach(func() {
		broker = NewServiceBroker(
			random_name.CATSRandomName("BROKER"),
			assets.NewAssets().ServiceBroker,
			context,
		)
		broker.Push(config)
		broker.Configure()
		broker.Create()
		broker.PublicizePlans()

		orgName = random_name.CATSRandomName("ORG")
		quotaName = random_name.CATSRandomName("QUOTA")
		spaceName := random_name.CATSRandomName("SPACE")
		appName := random_name.CATSRandomName("APP")
		instanceName := random_name.CATSRandomName("SVCINS")

		workflowhelpers.AsUser(UserContext.AdminUserContext(), DEFAULT_TIMEOUT, func() {
			createQuota := cf.Cf("create-quota", quotaName, "-m", "10G", "-r", "1000", "-s", "5").Wait(context.ShortTimeout())
			Expect(createQuota).To(Exit(0))

			createOrg := cf.Cf("create-org", orgName).Wait(DEFAULT_TIMEOUT)
			Expect(createOrg).To(Exit(0), "failed to create org")

			setQuota := cf.Cf("set-quota", orgName, quotaName).Wait(context.ShortTimeout())
			Expect(setQuota).To(Exit(0))

			createSpace := cf.Cf("create-space", spaceName, "-o", orgName).Wait(DEFAULT_TIMEOUT)
			Expect(createSpace).To(Exit(0), "failed to create space")

			target := cf.Cf("target", "-o", orgName, "-s", spaceName).Wait(CF_PUSH_TIMEOUT)
			Expect(target).To(Exit(0), "failed targeting")

			createApp := cf.Cf("push", appName, "--no-start", "-b", Config.RubyBuildpackName, "-m", DEFAULT_MEMORY_LIMIT, "-p", assets.NewAssets().Dora, "-d", Config.AppsDomain).Wait(DEFAULT_TIMEOUT)
			Expect(createApp).To(Exit(0), "failed creating app")
			app_helpers.SetBackend(appName)
			Expect(cf.Cf("start", appName).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

			createService := cf.Cf("create-service", broker.Service.Name, broker.SyncPlans[0].Name, instanceName).Wait(DEFAULT_TIMEOUT)
			Expect(createService).To(Exit(0), "failed creating service")
		})
	})

	AfterEach(func() {
		app_helpers.AppReport(broker.Name, DEFAULT_TIMEOUT)

		broker.Destroy()
		workflowhelpers.AsUser(UserContext.AdminUserContext(), DEFAULT_TIMEOUT, func() {
			deleteQuota := cf.Cf("delete-quota", "-f", quotaName).Wait(context.ShortTimeout())
			Expect(deleteQuota).To(Exit(0))
		})
	})

	It("deletes all apps and services in all spaces in an org", func() {
		workflowhelpers.AsUser(UserContext.AdminUserContext(), DEFAULT_TIMEOUT, func() {
			deleteOrg := cf.Cf("delete-org", orgName, "-f").Wait(DEFAULT_TIMEOUT)
			Expect(deleteOrg).To(Exit(0), "failed deleting org")
		})
		getOrg := cf.Cf("org", orgName).Wait(DEFAULT_TIMEOUT)
		Expect(getOrg).To(Exit(1), "org still exists")
	})
})
