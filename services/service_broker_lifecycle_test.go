package services

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/vito/cmdtest/matchers"

	. "github.com/cloudfoundry/cf-acceptance-tests/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/services/helpers"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
	"github.com/pivotal-cf-experimental/cf-test-helpers/generator"
)

var _ = Describe("Service Broker Lifecycle", func() {
	var broker helpers.ServiceBroker

	BeforeEach(func() {
		broker = helpers.NewServiceBroker(generator.RandomName(), NewAssets().ServiceBroker)
		broker.Push()
		broker.Configure()
	})

	It("confirms correct behavior in the lifecycle of a service broker", func() {
		// Adding the service broker
		broker.Create(LoadConfig().AppsDomain)

		// Confirming the plans are not yet public
		session := Cf("marketplace")
		Expect(session).NotTo(Say(broker.Service.Name))
		Expect(session).NotTo(Say(broker.Plan.Name))

		broker.PublicizePlans()

		// Confirming plans show up in the marketplace
		session = Cf("marketplace")
		Expect(session).To(Say(broker.Service.Name))
		Expect(session).To(Say(broker.Plan.Name))

		// Changing the catalog on the broker
		oldServiceName := broker.Service.Name
		oldPlanName := broker.Plan.Name
		broker.Service.Name = generator.RandomName()
		broker.Plan.Name = generator.RandomName()
		broker.Configure()
		broker.Update(LoadConfig().AppsDomain)

		// Confirming the changes to the broker show up in the marketplace
		session = Cf("marketplace")
		Expect(session).NotTo(Say(oldServiceName))
		Expect(session).NotTo(Say(oldPlanName))
		Expect(session).To(Say(broker.Service.Name))
		Expect(session).To(Say(broker.Plan.Name))

		// Deleting the service broker and confirming the plans no longer display
		broker.Delete()
		session = Cf("marketplace")
		Expect(session).NotTo(Say(oldServiceName))
		Expect(session).NotTo(Say(oldPlanName))
		Expect(session).NotTo(Say(broker.Service.Name))
		Expect(session).NotTo(Say(broker.Plan.Name))

		broker.Destroy()
	})
})
