package services

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
)

var _ = Describe("Service Broker Lifecycle", func() {
	var broker ServiceBroker
	var acls *Session
	var output []byte
	var oldServiceName string
	var oldPlanName string

	BeforeEach(func() {
		cf.AsUser(context.RegularUserContext(), context.ShortTimeout(), func() {
			broker = NewServiceBroker(generator.RandomName(), assets.NewAssets().ServiceBroker, context)
			cf.TargetSpace(context.RegularUserContext(), context.ShortTimeout())
			broker.Push()
			broker.Configure()
		})

		cf.AsUser(context.AdminUserContext(), context.ShortTimeout(), func() {
			broker.Create()
		})
	})

	Describe("Updating the catalog", func() {

		BeforeEach(func() {
			broker.PublicizePlans()
		})

		It("updates the broker and sees catalog changes", func() {
			// Confirming plans show up in the marketplace for regular user
			cf.AsUser(context.RegularUserContext(), context.ShortTimeout(), func() {
				plans := cf.Cf("marketplace").Wait(DEFAULT_TIMEOUT)
				Expect(plans).To(Exit(0))
				output = plans.Out.Contents()
				Expect(output).To(ContainSubstring(broker.Service.Name))

				Expect(output).To(ContainSubstring(broker.SyncPlans[0].Name))
				Expect(output).To(ContainSubstring(broker.SyncPlans[1].Name))

				// Changing the catalog on the broker
				oldServiceName = broker.Service.Name
				oldPlanName = broker.SyncPlans[0].Name
				broker.Service.Name = generator.RandomName()
				broker.SyncPlans[0].Name = generator.RandomName()
				broker.Configure()
				broker.Update()
			})

			// Confirming the changes to the broker show up in the marketplace
			cf.AsUser(context.RegularUserContext(), context.ShortTimeout(), func() {
				plans := cf.Cf("marketplace").Wait(DEFAULT_TIMEOUT)
				Expect(plans).To(Exit(0))
				output = plans.Out.Contents()
				Expect(output).NotTo(ContainSubstring(oldServiceName))
				Expect(output).NotTo(ContainSubstring(oldPlanName))
				Expect(output).To(ContainSubstring(broker.Service.Name))
				Expect(output).To(ContainSubstring(broker.Plans()[0].Name))
			})

			// Deleting the service broker and confirming the plans no longer display
			cf.AsUser(context.AdminUserContext(), context.ShortTimeout(), func() {
				broker.Delete()
			})

			cf.AsUser(context.RegularUserContext(), context.ShortTimeout(), func() {
				plans := cf.Cf("marketplace").Wait(DEFAULT_TIMEOUT)
				Expect(plans).To(Exit(0))
				output = plans.Out.Contents()
				Expect(output).NotTo(ContainSubstring(oldServiceName))
				Expect(output).NotTo(ContainSubstring(oldPlanName))
				Expect(output).NotTo(ContainSubstring(broker.Service.Name))
				Expect(output).NotTo(ContainSubstring(broker.Plans()[0].Name))
			})
		})
	})

	Describe("service access", func() {
		var (
			accessOutput        = "%[1]s\\s{2,}%[2]s\\s{2,}%[3]s\\s*\n"
			accessOutputWithOrg = "%[1]s\\s{2,}%[2]s\\s{2,}%[3]s\\s{2,}%[4]s\\s*"
			globallyPublicPlan  Plan
			orgPublicPlan       Plan
		)

		BeforeEach(func() {
			globallyPublicPlan = broker.Plans()[0]
			orgPublicPlan = broker.Plans()[1]
		})

		Describe("enabling", func() {
			It("is visible to a regular user", func() {
				cf.AsUser(context.AdminUserContext(), context.ShortTimeout(), func() {
					commandResult := cf.Cf("enable-service-access", broker.Service.Name, "-p", globallyPublicPlan.Name).Wait(DEFAULT_TIMEOUT)
					Expect(commandResult).To(Exit(0))
					commandResult = cf.Cf("enable-service-access", broker.Service.Name, "-p", orgPublicPlan.Name, "-o", context.RegularUserContext().Org).Wait(DEFAULT_TIMEOUT)
					Expect(commandResult).To(Exit(0))
				})
				cf.AsUser(context.RegularUserContext(), context.ShortTimeout(), func() {
					plans := cf.Cf("marketplace").Wait(DEFAULT_TIMEOUT)
					Expect(plans).To(Exit(0))
					output = plans.Out.Contents()
					Expect(output).To(ContainSubstring(broker.Service.Name))

					Expect(output).To(ContainSubstring(globallyPublicPlan.Name))
					Expect(output).To(ContainSubstring(orgPublicPlan.Name))
				})
			})

			It("is visible to an admin user", func() {
				cf.AsUser(context.AdminUserContext(), context.ShortTimeout(), func() {
					commandResult := cf.Cf("enable-service-access", broker.Service.Name, "-p", globallyPublicPlan.Name).Wait(DEFAULT_TIMEOUT)
					Expect(commandResult).To(Exit(0))
					commandResult = cf.Cf("enable-service-access", broker.Service.Name, "-p", orgPublicPlan.Name, "-o", context.RegularUserContext().Org).Wait(DEFAULT_TIMEOUT)
					Expect(commandResult).To(Exit(0))
				})

				cf.AsUser(context.AdminUserContext(), context.ShortTimeout(), func() {
					acls = cf.Cf("service-access", "-e", broker.Service.Name).Wait(DEFAULT_TIMEOUT)
					Expect(acls).To(Exit(0))
					output = acls.Out.Contents()
					Expect(output).To(ContainSubstring(broker.Service.Name))

					expectedOutput := fmt.Sprintf(accessOutput, broker.Service.Name, globallyPublicPlan.Name, "all")
					Expect(output).To(MatchRegexp(expectedOutput))

					expectedOutput = fmt.Sprintf(accessOutputWithOrg, broker.Service.Name, orgPublicPlan.Name, "limited", context.RegularUserContext().Org)
					Expect(output).To(MatchRegexp(expectedOutput))
				})
			})
		})

		Describe("disabling", func() {

			BeforeEach(func() {
				cf.AsUser(context.AdminUserContext(), context.ShortTimeout(), func() {
					commandResult := cf.Cf("enable-service-access", broker.Service.Name, "-p", globallyPublicPlan.Name).Wait(DEFAULT_TIMEOUT)
					Expect(commandResult).To(Exit(0))
					commandResult = cf.Cf("enable-service-access", broker.Service.Name, "-p", orgPublicPlan.Name, "-o", context.RegularUserContext().Org).Wait(DEFAULT_TIMEOUT)
					Expect(commandResult).To(Exit(0))
				})
			})

			It("is not visible to a regular user", func() {
				cf.AsUser(context.AdminUserContext(), context.ShortTimeout(), func() {
					commandResult := cf.Cf("disable-service-access", broker.Service.Name, "-p", orgPublicPlan.Name, "-o", context.RegularUserContext().Org).Wait(DEFAULT_TIMEOUT)
					Expect(commandResult).To(Exit(0))
					commandResult = cf.Cf("disable-service-access", broker.Service.Name, "-p", globallyPublicPlan.Name).Wait(DEFAULT_TIMEOUT)
					Expect(commandResult).To(Exit(0))
				})

				cf.AsUser(context.RegularUserContext(), context.ShortTimeout(), func() {
					plans := cf.Cf("marketplace").Wait(DEFAULT_TIMEOUT)
					Expect(plans).To(Exit(0))
					output = plans.Out.Contents()
					Expect(output).NotTo(ContainSubstring(broker.Service.Name))

					Expect(output).NotTo(ContainSubstring(globallyPublicPlan.Name))
					Expect(output).NotTo(ContainSubstring(orgPublicPlan.Name))
				})
			})

			It("is visible as having no access to an admin user", func() {
				cf.AsUser(context.AdminUserContext(), context.ShortTimeout(), func() {
					commandResult := cf.Cf("disable-service-access", broker.Service.Name, "-p", orgPublicPlan.Name, "-o", context.RegularUserContext().Org).Wait(DEFAULT_TIMEOUT)
					Expect(commandResult).To(Exit(0))
					commandResult = cf.Cf("disable-service-access", broker.Service.Name, "-p", globallyPublicPlan.Name).Wait(DEFAULT_TIMEOUT)
					Expect(commandResult).To(Exit(0))
					acls = cf.Cf("service-access", "-e", broker.Service.Name).Wait(DEFAULT_TIMEOUT)
					Expect(acls).To(Exit(0))
					output = acls.Out.Contents()
					Expect(output).To(ContainSubstring(broker.Service.Name))

					expectedOutput := fmt.Sprintf(accessOutput, broker.Service.Name, globallyPublicPlan.Name, "none")
					Expect(output).To(MatchRegexp(expectedOutput))
					expectedOutput = fmt.Sprintf(accessOutput, broker.Service.Name, orgPublicPlan.Name, "none")
					Expect(output).To(MatchRegexp(expectedOutput))
				})
			})
		})
	})

	AfterEach(func() {
		broker.Destroy()
	})
})
