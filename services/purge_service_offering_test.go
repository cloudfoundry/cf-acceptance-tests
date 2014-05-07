package services

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers"
	shelpers "github.com/cloudfoundry/cf-acceptance-tests/services/helpers"
)

var _ = Describe("Purging service offerings", func() {
	var broker shelpers.ServiceBroker

	BeforeEach(func() {
		broker = shelpers.NewServiceBroker(generator.RandomName(), helpers.NewAssets().ServiceBroker, context)
		broker.Push()
		broker.Configure()
		broker.Create()
		broker.PublicizePlans()
	})

	AfterEach(func() {
		broker.Destroy()
	})

	It("removes all instances and plans of the service, then removes the service offering", func() {
		instanceName := "purge-offering-instance"

		Eventually(cf.Cf("marketplace"), DEFAULT_TIMEOUT).Should(Say(broker.Plan.Name))
		broker.CreateServiceInstance(instanceName)

		Eventually(cf.Cf("services"), DEFAULT_TIMEOUT).Should(Say(instanceName))
		Eventually(cf.Cf("delete", broker.Name, "-f"), DEFAULT_TIMEOUT).Should(Exit(0))
		cf.AsUser(context.AdminUserContext(), func() {
			Eventually(cf.Cf("purge-service-offering", broker.Service.Name, "-f"), DEFAULT_TIMEOUT).Should(Exit(0))
		})
		Expect(cf.Cf("services").Wait(DEFAULT_TIMEOUT).Out.Contents()).NotTo(ContainSubstring(instanceName))
		Expect(cf.Cf("marketplace").Wait(DEFAULT_TIMEOUT).Out.Contents()).NotTo(ContainSubstring(broker.Service.Name))
	})
})
