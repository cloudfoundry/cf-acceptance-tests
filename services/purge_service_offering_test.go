package services

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	. "github.com/cloudfoundry/cf-acceptance-tests/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/services/helpers"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
	"github.com/pivotal-cf-experimental/cf-test-helpers/generator"
)

var _ = Describe("Purging service offerings", func() {
	var broker helpers.ServiceBroker

	BeforeEach(func() {
		broker = helpers.NewServiceBroker(generator.RandomName(), NewAssets().ServiceBroker, context)
		broker.Push()
		broker.Configure()
		broker.Create(LoadConfig().AppsDomain)
		broker.PublicizePlans()
	})

	AfterEach(func() {
		broker.Destroy()
	})

	It("removes all instances and plans of the service, then removes the service offering", func() {
		instanceName := "purge-offering-instance"

		Eventually(Cf("marketplace"), DefaultTimeout).Should(Say(broker.Plan.Name))
		broker.CreateServiceInstance(instanceName)

		Eventually(Cf("services"), DefaultTimeout).Should(Say(instanceName))
		Eventually(Cf("delete", broker.Name, "-f"), DefaultTimeout).Should(Exit(0))
		AsUser(context.AdminUserContext(), func() {
			Eventually(Cf("purge-service-offering", broker.Service.Name, "-f"), DefaultTimeout).Should(Exit(0))
		})
		Expect(Cf("services").Wait(DefaultTimeout).Out.Contents()).NotTo(ContainSubstring(instanceName))
		Expect(Cf("marketplace").Wait(DefaultTimeout).Out.Contents()).NotTo(ContainSubstring(broker.Service.Name))
	})
})
