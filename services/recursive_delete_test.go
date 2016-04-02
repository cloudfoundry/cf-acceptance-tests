package services_test

import (
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/services"

	"github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"

	. "github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/onsi/ginkgo"
	. "github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/onsi/gomega"
	. "github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/onsi/gomega/gexec"
)

var _ = Describe("Recursive Delete", func() {
	var broker ServiceBroker
	var orgName string
	var quotaName string

	BeforeEach(func() {
		broker = NewServiceBroker(
			generator.PrefixedRandomName("rec-del"),
			assets.NewAssets().ServiceBroker,
			context,
		)
		broker.Push()
		broker.Configure()
		broker.Create()
		broker.PublicizePlans()

		orgName = generator.PrefixedRandomName("rec-del")
		quotaName = generator.PrefixedRandomName("rec-del")
		spaceName := generator.PrefixedRandomName("rec-del")
		appName := generator.PrefixedRandomName("rec-del")
		instanceName := generator.PrefixedRandomName("rec-del")

		cf.AsUser(context.AdminUserContext(), DEFAULT_TIMEOUT, func() {
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

			createApp := cf.Cf("push", appName, "--no-start", "-b", config.RubyBuildpackName, "-m", DEFAULT_MEMORY_LIMIT, "-p", assets.NewAssets().Dora, "-d", config.AppsDomain).Wait(DEFAULT_TIMEOUT)
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
		cf.AsUser(context.AdminUserContext(), DEFAULT_TIMEOUT, func() {
			deleteQuota := cf.Cf("delete-quota", "-f", quotaName).Wait(context.ShortTimeout())
			Expect(deleteQuota).To(Exit(0))
		})
	})

	It("deletes all apps and services in all spaces in an org", func() {
		cf.AsUser(context.AdminUserContext(), DEFAULT_TIMEOUT, func() {
			deleteOrg := cf.Cf("delete-org", orgName, "-f").Wait(DEFAULT_TIMEOUT)
			Expect(deleteOrg).To(Exit(0), "failed deleting org")
		})
		getOrg := cf.Cf("org", orgName).Wait(DEFAULT_TIMEOUT)
		Expect(getOrg).To(Exit(1), "org still exists")
	})
})
