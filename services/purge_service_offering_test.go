package services

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/vito/cmdtest/matchers"

	. "github.com/cloudfoundry/cf-acceptance-tests/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/services/helpers"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
	"github.com/pivotal-cf-experimental/cf-test-helpers/generator"
)

var _ = Describe("Purging service offerings", func() {
	var broker helpers.ServiceBroker

	BeforeEach(func() {
		broker = helpers.NewServiceBroker(generator.RandomName(), NewAssets().ServiceBroker)
		broker.Push()
		broker.Configure()
		broker.Create(LoadConfig().AppsDomain)
		broker.PublicizePlans()
	})

	AfterEach(func() {
		broker.Destroy()
	})

	It("removes all instances and plans of the service, then removes the service offering", func() {
		defer helpers.Recover() // Catches panic thrown by Require expectations

		instanceName := "purge-offering-instance"

		Expect(Cf("marketplace")).To(Say(broker.Plan.Name))
		broker.CreateServiceInstance(instanceName)

		Expect(Cf("services")).To(Say(instanceName))
		Expect(Cf("delete", broker.Name, "-f")).To(ExitWithTimeout(0, 10*time.Second))
		AsUser(AdminUserContext, func() {
			Expect(Cf("purge-service-offering", broker.Service.Name, "-f")).To(ExitWithTimeout(0, 10*time.Second))
		})
		Expect(Cf("services")).NotTo(Say(instanceName))
		Expect(Cf("marketplace")).NotTo(Say(broker.Service.Name))
	})
})
