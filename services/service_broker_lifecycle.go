package services_test

import (
	"encoding/json"
	"fmt"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/services"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/helpers"
	"github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = ServicesDescribe("Service Broker Lifecycle", func() {
	var broker ServiceBroker

	Describe("public brokers", func() {
		var acls *Session
		var output []byte
		var oldServiceName string
		var oldPlanName string
		var otherOrgName string

		BeforeEach(func() {
			broker = NewServiceBroker(
				random_name.CATSRandomName("BRKR"),
				assets.NewAssets().ServiceBroker,
				TestSetup,
			)
			TestSetup.RegularUserContext().TargetSpace()
			broker.Push(Config)
			broker.Configure()

			workflowhelpers.AsUser(TestSetup.AdminUserContext(), TestSetup.ShortTimeout(), func() {
				broker.Create()
			})
		})

		Describe("Updating the catalog", func() {
			BeforeEach(func() {
				broker.PublicizePlans()
			})

			It("updates the broker and sees catalog changes", func() {
				// Confirming plans show up in the marketplace for regular user
				plans := cf.Cf("marketplace").Wait()
				Expect(plans).To(Exit(0))
				Expect(plans).To(Say(broker.Service.Name))

				Expect(plans).To(Say(broker.SyncPlans[0].Name))
				Expect(plans).To(Say(broker.SyncPlans[1].Name))

				// Confirm default schemas show up in CAPI
				cfResponse := cf.Cf("curl", fmt.Sprintf("/v3/service_plans?broker_catalog_ids=%s", broker.SyncPlans[0].ID)).Wait().Out.Contents()

				var plansResponse ServicesPlansResponse
				err := json.Unmarshal(cfResponse, &plansResponse)
				Expect(err).To(BeNil())

				var emptySchemas PlanSchemas
				emptySchemas.ServiceInstance.Create.Parameters = map[string]interface{}{}
				emptySchemas.ServiceInstance.Update.Parameters = map[string]interface{}{}
				emptySchemas.ServiceBinding.Create.Parameters = map[string]interface{}{}

				Expect(plansResponse.Resources[0].Schemas).To(Equal(emptySchemas))

				// Changing the catalog on the broker
				oldServiceName = broker.Service.Name
				oldPlanName = broker.SyncPlans[0].Name
				broker.Service.Name = random_name.CATSRandomName("SVC")
				broker.SyncPlans[0].Name = random_name.CATSRandomName("SVC-PLAN")

				var basicSchema PlanSchemas
				basicSchema.ServiceInstance.Create.Parameters = map[string]interface{}{
					"$schema": "http://json-schema.org/draft-04/schema#",
					"type":    "object",
					"title":   "create instance schema",
				}
				basicSchema.ServiceInstance.Update.Parameters = map[string]interface{}{
					"$schema": "http://json-schema.org/draft-04/schema#",
					"type":    "object",
					"title":   "update instance schema",
				}
				basicSchema.ServiceBinding.Create.Parameters = map[string]interface{}{
					"$schema": "http://json-schema.org/draft-04/schema#",
					"type":    "object",
					"title":   "create binding schema",
				}
				broker.SyncPlans[0].Schemas = basicSchema

				broker.Configure()
				broker.Update()

				// Confirming the changes to the broker show up in the marketplace
				plans = cf.Cf("marketplace").Wait()
				Expect(plans).To(Exit(0))
				Expect(plans).NotTo(Say(oldServiceName))
				Expect(plans).NotTo(Say(oldPlanName))
				Expect(plans).To(Say(broker.Service.Name))
				Expect(plans).To(Say(broker.Plans()[0].Name))

				// Confirm plan schemas show up in CAPI
				cfResponse = cf.Cf("curl", fmt.Sprintf("/v3/service_plans?broker_catalog_ids=%s", broker.SyncPlans[0].ID)).Wait().Out.Contents()

				err = json.Unmarshal(cfResponse, &plansResponse)
				Expect(err).To(BeNil())
				Expect(plansResponse.Resources[0].Schemas).To(Equal(broker.SyncPlans[0].Schemas))

				// Deleting the service broker and confirming the plans no longer display
				workflowhelpers.AsUser(TestSetup.AdminUserContext(), TestSetup.ShortTimeout(), func() {
					broker.Delete()
				})

				plans = cf.Cf("marketplace").Wait()
				Expect(plans).To(Exit(0))
				Expect(plans).NotTo(Say(oldServiceName))
				Expect(plans).NotTo(Say(oldPlanName))
				Expect(plans).NotTo(Say(broker.Service.Name))
				Expect(plans).NotTo(Say(broker.Plans()[0].Name))
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
				BeforeEach(func() {
					workflowhelpers.AsUser(TestSetup.AdminUserContext(), TestSetup.ShortTimeout(), func() {
						otherOrgName = random_name.CATSRandomName("ORG")
						createOrg := cf.Cf("create-org", otherOrgName).Wait()
						Expect(createOrg).To(Exit(0), "failed to create org")

						addOrgManager := cf.Cf("set-org-role", TestSetup.RegularUserContext().Username, otherOrgName, "OrgManager").Wait()
						Expect(addOrgManager).To(Exit(0), "failed to add org manager role")

						commandResult := cf.Cf("enable-service-access", broker.Service.Name, "-p", globallyPublicPlan.Name).Wait()
						Expect(commandResult).To(Exit(0))
						commandResult = cf.Cf("enable-service-access", broker.Service.Name, "-p", orgPublicPlan.Name, "-o", TestSetup.RegularUserContext().Org).Wait()
						Expect(commandResult).To(Exit(0))
						commandResult = cf.Cf("enable-service-access", broker.Service.Name, "-p", orgPublicPlan.Name, "-o", otherOrgName).Wait()
						Expect(commandResult).To(Exit(0))
					})
				})
				It("is visible to a regular user", func() {
					var plansResponse ServicesPlansResponse
					workflowhelpers.ApiRequest("GET", fmt.Sprintf("/v3/service_plans?service_offering_names=%s", broker.Service.Name), &plansResponse, Config.DefaultTimeoutDuration())
					// Ensure that there no duplicates
					Expect(len(plansResponse.Resources)).To(Equal(2))
					Expect(plansResponse.Resources[0].Name).To(Equal(globallyPublicPlan.Name))
					Expect(plansResponse.Resources[1].Name).To(Equal(orgPublicPlan.Name))
					plans := cf.Cf("marketplace").Wait()
					Expect(plans).To(Exit(0))
					Expect(plans).To(Say(broker.Service.Name))

					Expect(plans).To(Say(globallyPublicPlan.Name))
					Expect(plans).To(Say(orgPublicPlan.Name))
				})

				It("is visible to an admin user", func() {
					workflowhelpers.AsUser(TestSetup.AdminUserContext(), TestSetup.ShortTimeout(), func() {
						acls = cf.Cf("service-access", "-e", broker.Service.Name).Wait()
						Expect(acls).To(Exit(0))
						output = acls.Out.Contents()
						Expect(output).To(ContainSubstring(broker.Service.Name))

						expectedOutput := fmt.Sprintf(accessOutput, broker.Service.Name, globallyPublicPlan.Name, "all")
						Expect(output).To(MatchRegexp(expectedOutput))

						expectedOutput = fmt.Sprintf(accessOutputWithOrg, broker.Service.Name, orgPublicPlan.Name, "limited", TestSetup.RegularUserContext().Org)
						Expect(output).To(MatchRegexp(expectedOutput))
					})
				})
			})

			Describe("disabling", func() {

				BeforeEach(func() {
					workflowhelpers.AsUser(TestSetup.AdminUserContext(), TestSetup.ShortTimeout(), func() {
						commandResult := cf.Cf("enable-service-access", broker.Service.Name, "-p", globallyPublicPlan.Name).Wait()
						Expect(commandResult).To(Exit(0))
						commandResult = cf.Cf("enable-service-access", broker.Service.Name, "-p", orgPublicPlan.Name, "-o", TestSetup.RegularUserContext().Org).Wait()
						Expect(commandResult).To(Exit(0))
					})
				})

				It("is not visible to a regular user", func() {
					workflowhelpers.AsUser(TestSetup.AdminUserContext(), TestSetup.ShortTimeout(), func() {
						commandResult := cf.Cf("disable-service-access", broker.Service.Name, "-p", orgPublicPlan.Name, "-o", TestSetup.RegularUserContext().Org).Wait()
						Expect(commandResult).To(Exit(0))
						commandResult = cf.Cf("disable-service-access", broker.Service.Name, "-p", globallyPublicPlan.Name).Wait()
						Expect(commandResult).To(Exit(0))
					})

					plans := cf.Cf("marketplace").Wait()
					Expect(plans).To(Exit(0))
					Expect(plans).NotTo(Say(broker.Service.Name))

					Expect(plans).NotTo(Say(globallyPublicPlan.Name))
					Expect(plans).NotTo(Say(orgPublicPlan.Name))
				})

				It("is visible as having no access to an admin user", func() {
					workflowhelpers.AsUser(TestSetup.AdminUserContext(), TestSetup.ShortTimeout(), func() {
						commandResult := cf.Cf("disable-service-access", broker.Service.Name, "-p", orgPublicPlan.Name, "-o", TestSetup.RegularUserContext().Org).Wait()
						Expect(commandResult).To(Exit(0))
						commandResult = cf.Cf("disable-service-access", broker.Service.Name, "-p", globallyPublicPlan.Name).Wait()
						Expect(commandResult).To(Exit(0))
						acls = cf.Cf("service-access", "-e", broker.Service.Name).Wait()
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
			app_helpers.AppReport(broker.Name)

			broker.Destroy()
		})
	})

	Describe("space scoped brokers", func() {
		BeforeEach(func() {
			broker = NewServiceBroker(
				random_name.CATSRandomName("BRKR"),
				assets.NewAssets().ServiceBroker,
				TestSetup,
			)
			TestSetup.RegularUserContext().TargetSpace()
			broker.Push(Config)
			broker.Configure()
		})

		AfterEach(func() {
			app_helpers.AppReport(broker.Name)
			broker.Destroy()
		})

		It("can be created, viewed (in list), updated, and deleted by SpaceDevelopers", func() {
			workflowhelpers.AsUser(TestSetup.RegularUserContext(), TestSetup.ShortTimeout(), func() {
				By("Create")
				createBrokerCommand := cf.Cf("create-service-broker",
					broker.Name,
					TestSetup.RegularUserContext().Username,
					TestSetup.RegularUserContext().Password,
					helpers.AppUri(broker.Name, "", Config),
					"--space-scoped",
				).Wait()
				Expect(createBrokerCommand).To(Exit(0))

				By("Read")
				serviceBrokersCommand := cf.Cf("service-brokers").Wait()
				Expect(serviceBrokersCommand).To(Exit(0))
				Expect(serviceBrokersCommand.Out.Contents()).To(ContainSubstring(broker.Name))

				By("Update")
				updatedName := broker.Name + "updated"
				updateBrokerCommand := cf.Cf("rename-service-broker", broker.Name, updatedName).Wait()
				Expect(updateBrokerCommand).To(Exit(0))

				serviceBrokersCommand = cf.Cf("service-brokers").Wait()
				Expect(serviceBrokersCommand).To(Exit(0))
				Expect(serviceBrokersCommand.Out.Contents()).To(ContainSubstring(updatedName))

				By("Delete")
				deleteBrokerCommand := cf.Cf("delete-service-broker", updatedName, "-f").Wait()
				Expect(deleteBrokerCommand).To(Exit(0))

				serviceBrokersCommand = cf.Cf("service-brokers").Wait()
				Expect(serviceBrokersCommand).To(Exit(0))
				Expect(serviceBrokersCommand.Out.Contents()).NotTo(ContainSubstring(updatedName))
			})
		})

		It("exposes the services and plans of the private broker in the space", func() {
			workflowhelpers.AsUser(TestSetup.RegularUserContext(), TestSetup.ShortTimeout(), func() {
				createBrokerCommand := cf.Cf("create-service-broker",
					broker.Name,
					TestSetup.RegularUserContext().Username,
					TestSetup.RegularUserContext().Password,
					helpers.AppUri(broker.Name, "", Config),
					"--space-scoped",
				).Wait()
				Expect(createBrokerCommand).To(Exit(0))

				marketplaceOutput := cf.Cf("marketplace").Wait()

				Expect(marketplaceOutput).To(Exit(0))
				Expect(marketplaceOutput).To(Say(broker.Service.Name))
				Expect(marketplaceOutput).To(Say(broker.SyncPlans[0].Name))
				Expect(marketplaceOutput).To(Say(broker.SyncPlans[1].Name))
			})
		})
	})
})
