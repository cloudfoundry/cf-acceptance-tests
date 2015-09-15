package services_test

import (
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/services"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/runner"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Recursive Delete", func() {
	var broker ServiceBroker
	var orgName string
	var quotaName string

	BeforeEach(func() {
		broker = NewServiceBroker(generator.RandomName(), assets.NewAssets().ServiceBroker, context)
		broker.Push()
		broker.Configure()
		broker.Create()
		broker.PublicizePlans()

		orgName = generator.RandomName()
		quotaName = generator.RandomName() + "-recursive-delete"
		spaceName := generator.RandomName()
		appName := generator.RandomName()
		instanceName := generator.RandomName()

		cf.AsUser(context.AdminUserContext(), DEFAULT_TIMEOUT, func() {

			runner.NewCmdRunner(cf.Cf("create-quota", quotaName, "-m", "10G", "-r", "1000", "-s", "5"), context.ShortTimeout()).Run()

			createOrg := cf.Cf("create-org", orgName).Wait(DEFAULT_TIMEOUT)
			Expect(createOrg).To(Exit(0), "failed to create org")

			runner.NewCmdRunner(cf.Cf("set-quota", orgName, quotaName), context.ShortTimeout()).Run()

			createSpace := cf.Cf("create-space", spaceName, "-o", orgName).Wait(DEFAULT_TIMEOUT)
			Expect(createSpace).To(Exit(0), "failed to create space")

			target := cf.Cf("target", "-o", orgName, "-s", spaceName).Wait(CF_PUSH_TIMEOUT)
			Expect(target).To(Exit(0), "failed targeting")

			createApp := cf.Cf("push", appName, "-m", "128M", "-p", assets.NewAssets().Dora, "-d", config.AppsDomain).Wait(CF_PUSH_TIMEOUT)
			Expect(createApp).To(Exit(0), "failed creating app")

			createService := cf.Cf("create-service", broker.Service.Name, broker.SyncPlans[0].Name, instanceName).Wait(DEFAULT_TIMEOUT)
			Expect(createService).To(Exit(0), "failed creating service")
		})
	})

	AfterEach(func() {
		broker.Destroy()
		runner.NewCmdRunner(cf.Cf("delete-quota", quotaName), context.ShortTimeout()).Run()
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
