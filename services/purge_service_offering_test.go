package services_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/services"
)

var _ = Describe("Purging service offerings", func() {
	var broker ServiceBroker

	BeforeEach(func() {
		broker = NewServiceBroker(generator.RandomName(), assets.NewAssets().ServiceBroker, context)
		broker.Push()
		broker.Configure()
		broker.Create()
		broker.PublicizePlans()
	})

	AfterEach(func() {
		broker.Destroy()
	})

	Context("when there are several existing service entities", func() {
		var appName, instanceName string

		BeforeEach(func() {
			appName = generator.RandomName()
			instanceName = generator.RandomName()

			createApp := cf.Cf("push", appName, "-p", assets.NewAssets().Dora).Wait(CF_PUSH_TIMEOUT)
			Expect(createApp).To(Exit(0), "failed creating app")

			broker.CreateServiceInstance(instanceName)

			services := cf.Cf("services").Wait(DEFAULT_TIMEOUT)
			Expect(services).To(Exit(0))
			Expect(services).To(Say(instanceName))

			bindService := cf.Cf("bind-service", appName, instanceName).Wait(DEFAULT_TIMEOUT)
			Expect(bindService).To(Exit(0), "failed binding app to service")
		})

		It("removes all instances and plans of the service, then removes the service offering", func() {
			marketplace := cf.Cf("marketplace").Wait(DEFAULT_TIMEOUT)
			Expect(marketplace).To(Exit(0))
			Expect(marketplace).To(Say(broker.Plans()[0].Name))

			Expect(cf.Cf("delete", broker.Name, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))

			cf.AsUser(context.AdminUserContext(), DEFAULT_TIMEOUT, func() {
				Expect(cf.Cf("purge-service-offering", broker.Service.Name, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			})

			services := cf.Cf("services").Wait(DEFAULT_TIMEOUT)
			Expect(services).To(Exit(0))
			Expect(services).NotTo(Say(instanceName))
			Expect(services).NotTo(Say(appName))

			marketplace = cf.Cf("marketplace").Wait(DEFAULT_TIMEOUT)
			Expect(marketplace).To(Exit(0))
			Expect(marketplace).NotTo(Say(broker.Service.Name))
		})
	})
})
