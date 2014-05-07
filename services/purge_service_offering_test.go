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

		marketplace := cf.Cf("marketplace").Wait(DEFAULT_TIMEOUT)
		Expect(marketplace).To(Exit(0))
		Expect(marketplace).To(Say(broker.Plan.Name))

		broker.CreateServiceInstance(instanceName)

		services := cf.Cf("services").Wait(DEFAULT_TIMEOUT)
		Expect(marketplace).To(Exit(0))
		Expect(services).To(Say(instanceName))

		Expect(cf.Cf("delete", broker.Name, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))

		cf.AsUser(context.AdminUserContext(), func() {
			Expect(cf.Cf("purge-service-offering", broker.Service.Name, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		})

		services = cf.Cf("services").Wait(DEFAULT_TIMEOUT)
		Expect(services).To(Exit(0))
		Expect(services.Out.Contents()).NotTo(ContainSubstring(instanceName)) //TODO: Say?

		marketplace = cf.Cf("marketplace").Wait(DEFAULT_TIMEOUT)
		Expect(marketplace).To(Exit(0))
		Expect(marketplace.Out.Contents()).NotTo(ContainSubstring(broker.Service.Name)) //TODO: Say?
	})
})
