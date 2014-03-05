package services

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/vito/cmdtest/matchers"

	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/services"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
	"github.com/pivotal-cf-experimental/cf-test-helpers/generator"
)

var _ = Describe("Service Broker Lifecycle", func() {
	var broker ServiceBroker

	BeforeEach(func() {
		LoginAsAdmin()
		broker = NewServiceBroker(generator.RandomName())
		broker.Push()
		broker.Configure()
	})

	AfterEach(func() {
		LoginAsUser()
	})

	It("confirms correct behavior in the lifecycle of a service broker", func() {
		defer Recover() // Catches panic thrown by Require expectations

		// Adding the service broker
		Require(Cf("create-service-broker", broker.Name, "username", "password", AppUri(broker.Name, ""))).To(ExitWithTimeout(0, 10*time.Second))
		Expect(Cf("service-brokers")).To(Say(broker.Name))

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
		Require(Cf("update-service-broker", broker.Name, "username", "password", AppUri(broker.Name, ""))).To(ExitWithTimeout(0, 10*time.Second))

		// Confirming the changes to the broker show up in the marketplace
		session = Cf("marketplace")
		Expect(session).NotTo(Say(oldServiceName))
		Expect(session).NotTo(Say(oldPlanName))
		Expect(session).To(Say(broker.Service.Name))
		Expect(session).To(Say(broker.Plan.Name))

		// Deleting the service broker and confirming the plans no longer display
		Require(Cf("delete-service-broker", broker.Name, "-f")).To(ExitWithTimeout(0, 2*time.Second))
		session = Cf("marketplace")
		Expect(session).NotTo(Say(oldServiceName))
		Expect(session).NotTo(Say(oldPlanName))
		Expect(session).NotTo(Say(broker.Service.Name))
		Expect(session).NotTo(Say(broker.Plan.Name))

		Require(Cf("delete", b.Name, "-f")).To(ExitWithTimeout(0, 2*time.Second))
	})
})
