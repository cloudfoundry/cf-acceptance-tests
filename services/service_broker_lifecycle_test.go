package services

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/cf-acceptance-tests/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/services/helpers"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
	"github.com/pivotal-cf-experimental/cf-test-helpers/generator"
)

var _ = Describe("Service Broker Lifecycle", func() {
	var broker helpers.ServiceBroker

	BeforeEach(func() {
		broker = helpers.NewServiceBroker(generator.RandomName(), NewAssets().ServiceBroker, context)
		broker.Push()
		broker.Configure()
	})

	It("confirms correct behavior in the lifecycle of a service broker", func() {
		// Adding the service broker
		broker.Create(LoadConfig().AppsDomain)

		// Confirming the plans are not yet public
		plans := Cf("marketplace").Wait(DefaultTimeout).Out.Contents()
		Expect(plans).NotTo(ContainSubstring(broker.Service.Name))
		Expect(plans).NotTo(ContainSubstring(broker.Plan.Name))

		broker.PublicizePlans()

		// Confirming plans show up in the marketplace
		plans = Cf("marketplace").Wait(DefaultTimeout).Out.Contents()
		Expect(plans).To(ContainSubstring(broker.Service.Name))
		Expect(plans).To(ContainSubstring(broker.Plan.Name))

		// Changing the catalog on the broker
		oldServiceName := broker.Service.Name
		oldPlanName := broker.Plan.Name
		broker.Service.Name = generator.RandomName()
		broker.Plan.Name = generator.RandomName()
		broker.Configure()
		broker.Update(LoadConfig().AppsDomain)

		// Confirming the changes to the broker show up in the marketplace
		plans = Cf("marketplace").Wait(DefaultTimeout).Out.Contents()
		Expect(plans).NotTo(ContainSubstring(oldServiceName))
		Expect(plans).NotTo(ContainSubstring(oldPlanName))
		Expect(plans).To(ContainSubstring(broker.Service.Name))
		Expect(plans).To(ContainSubstring(broker.Plan.Name))

		// Deleting the service broker and confirming the plans no longer display
		broker.Delete()
		plans = Cf("marketplace").Wait(DefaultTimeout).Out.Contents()
		Expect(plans).NotTo(ContainSubstring(oldServiceName))
		Expect(plans).NotTo(ContainSubstring(oldPlanName))
		Expect(plans).NotTo(ContainSubstring(broker.Service.Name))
		Expect(plans).NotTo(ContainSubstring(broker.Plan.Name))

		broker.Destroy()
	})
})
