package services

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/vito/cmdtest/matchers"

	. "github.com/pivotal-cf-experimental/cf-acceptance-tests/helpers/services"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
	"github.com/pivotal-cf-experimental/cf-test-helpers/generator"
)

var _ = Describe("Purging service offerings", func() {
	var broker ServiceBroker

	BeforeEach(func() {
		LoginAsAdmin()

		broker = NewServiceBroker(generator.RandomName())
		broker.Push()
		broker.Configure()
		broker.Create()
		broker.PublicizePlans()
	})

	AfterEach(func() {
		broker.Destroy()
		LoginAsUser()
	})

	It("removes all instances and plans of the service, then removes the service offering", func() {
		defer Recover() // Catches panic thrown by Require expectations

		instanceName := "purge-offering-instance"

		Expect(Cf("marketplace")).To(Say(broker.Plan.Name))
		// TODO:  CreateServiceInstance is used as a workaround for the problem in cf 6.0.1 that prevents us from
		//        creating an instance of a service when there are more than 50 services in the environment.
		//        Should be replaced by the following line ASAP

		// Expect(Cf("create-service", broker.Service.Name, broker.Plan.Name, instanceName)).To(ExitWith(0))
		broker.CreateServiceInstance(instanceName)

		Expect(Cf("services")).To(Say(instanceName))
		Expect(Cf("delete", broker.Name, "-f")).To(ExitWithTimeout(0, 10*time.Second))
		Expect(Cf("purge-service-offering", broker.Service.Name, "-f")).To(ExitWithTimeout(0, 10*time.Second))
		Expect(Cf("services")).NotTo(Say(instanceName))
		Expect(Cf("marketplace")).NotTo(Say(broker.Service.Name))
	})
})
