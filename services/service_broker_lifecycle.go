package services_test

import (
	"encoding/json"
	"fmt"
	"strings"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/services"
)

var _ = ServicesDescribe("Service Broker Lifecycle", func() {
	var broker ServiceBroker

	Describe("public brokers", func() {
		var acls *Session
		var output []byte
		var oldServiceName string
		var oldPlanName string

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
				plans := cf.Cf("marketplace").Wait(Config.DefaultTimeoutDuration())
				Expect(plans).To(Exit(0))
				Expect(plans).To(Say(broker.Service.Name))

				Expect(plans).To(Say(broker.SyncPlans[0].Name))
				Expect(plans).To(Say(broker.SyncPlans[1].Name))

				// Confirm default schemas show up in CAPI
				cfResponse := cf.Cf("curl", fmt.Sprintf("/v2/service_plans?q=unique_id:%s", broker.SyncPlans[0].ID)).
					Wait(Config.DefaultTimeoutDuration()).Out.Contents()

				var plansResponse ServicePlansResponse
				err := json.Unmarshal(cfResponse, &plansResponse)
				Expect(err).To(BeNil())

				var emptySchemas PlanSchemas
				emptySchemas.ServiceInstance.Create.Parameters = map[string]interface{}{}
				emptySchemas.ServiceInstance.Update.Parameters = map[string]interface{}{}
				emptySchemas.ServiceBinding.Create.Parameters = map[string]interface{}{}

				Expect(plansResponse.Resources[0].Entity.Schemas).To(Equal(emptySchemas))

				// Changing the catalog on the broker
				oldServiceName = broker.Service.Name
				oldPlanName = broker.SyncPlans[0].Name
				broker.Service.Name = random_name.CATSRandomName("SVC")
				broker.SyncPlans[0].Name = random_name.CATSRandomName("SVC-PLAN")

				var basicSchema PlanSchemas
				basicSchema.ServiceInstance.Create.Parameters = map[string]interface{}{
					"$schema": "http://json-schema.org/draft-04/schema#",
					"title":   "create instance schema",
				}
				basicSchema.ServiceInstance.Update.Parameters = map[string]interface{}{
					"$schema": "http://json-schema.org/draft-04/schema#",
					"title":   "update instance schema",
				}
				basicSchema.ServiceBinding.Create.Parameters = map[string]interface{}{
					"$schema": "http://json-schema.org/draft-04/schema#",
					"title":   "create binding schema",
				}
				broker.SyncPlans[0].Schemas = basicSchema

				broker.Configure()
				broker.Update()

				// Confirming the changes to the broker show up in the marketplace
				plans = cf.Cf("marketplace").Wait(Config.DefaultTimeoutDuration())
				Expect(plans).To(Exit(0))
				Expect(plans).NotTo(Say(oldServiceName))
				Expect(plans).NotTo(Say(oldPlanName))
				Expect(plans).To(Say(broker.Service.Name))
				Expect(plans).To(Say(broker.Plans()[0].Name))

				// Confirm plan schemas show up in CAPI
				cfResponse = cf.Cf("curl", fmt.Sprintf("/v2/service_plans?q=unique_id:%s", broker.SyncPlans[0].ID)).
					Wait(Config.DefaultTimeoutDuration()).Out.Contents()

				err = json.Unmarshal(cfResponse, &plansResponse)
				Expect(err).To(BeNil())
				Expect(plansResponse.Resources[0].Entity.Schemas).To(Equal(broker.SyncPlans[0].Schemas))

				// Deleting the service broker and confirming the plans no longer display
				workflowhelpers.AsUser(TestSetup.AdminUserContext(), TestSetup.ShortTimeout(), func() {
					broker.Delete()
				})

				plans = cf.Cf("marketplace").Wait(Config.DefaultTimeoutDuration())
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
				It("is visible to a regular user", func() {
					workflowhelpers.AsUser(TestSetup.AdminUserContext(), TestSetup.ShortTimeout(), func() {
						commandResult := cf.Cf("enable-service-access", broker.Service.Name, "-p", globallyPublicPlan.Name).Wait(Config.DefaultTimeoutDuration())
						Expect(commandResult).To(Exit(0))
						commandResult = cf.Cf("enable-service-access", broker.Service.Name, "-p", orgPublicPlan.Name, "-o", TestSetup.RegularUserContext().Org).Wait(Config.DefaultTimeoutDuration())
						Expect(commandResult).To(Exit(0))
					})

					plans := cf.Cf("marketplace").Wait(Config.DefaultTimeoutDuration())
					Expect(plans).To(Exit(0))
					Expect(plans).To(Say(broker.Service.Name))

					Expect(plans).To(Say(globallyPublicPlan.Name))
					Expect(plans).To(Say(orgPublicPlan.Name))
				})

				It("is visible to an admin user", func() {
					workflowhelpers.AsUser(TestSetup.AdminUserContext(), TestSetup.ShortTimeout(), func() {
						commandResult := cf.Cf("enable-service-access", broker.Service.Name, "-p", globallyPublicPlan.Name).Wait(Config.DefaultTimeoutDuration())
						Expect(commandResult).To(Exit(0))
						commandResult = cf.Cf("enable-service-access", broker.Service.Name, "-p", orgPublicPlan.Name, "-o", TestSetup.RegularUserContext().Org).Wait(Config.DefaultTimeoutDuration())
						Expect(commandResult).To(Exit(0))

						acls = cf.Cf("service-access", "-e", broker.Service.Name).Wait(Config.DefaultTimeoutDuration())
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
						commandResult := cf.Cf("enable-service-access", broker.Service.Name, "-p", globallyPublicPlan.Name).Wait(Config.DefaultTimeoutDuration())
						Expect(commandResult).To(Exit(0))
						commandResult = cf.Cf("enable-service-access", broker.Service.Name, "-p", orgPublicPlan.Name, "-o", TestSetup.RegularUserContext().Org).Wait(Config.DefaultTimeoutDuration())
						Expect(commandResult).To(Exit(0))
					})
				})

				It("is not visible to a regular user", func() {
					workflowhelpers.AsUser(TestSetup.AdminUserContext(), TestSetup.ShortTimeout(), func() {
						commandResult := cf.Cf("disable-service-access", broker.Service.Name, "-p", orgPublicPlan.Name, "-o", TestSetup.RegularUserContext().Org).Wait(Config.DefaultTimeoutDuration())
						Expect(commandResult).To(Exit(0))
						commandResult = cf.Cf("disable-service-access", broker.Service.Name, "-p", globallyPublicPlan.Name).Wait(Config.DefaultTimeoutDuration())
						Expect(commandResult).To(Exit(0))
					})

					plans := cf.Cf("marketplace").Wait(Config.DefaultTimeoutDuration())
					Expect(plans).To(Exit(0))
					Expect(plans).NotTo(Say(broker.Service.Name))

					Expect(plans).NotTo(Say(globallyPublicPlan.Name))
					Expect(plans).NotTo(Say(orgPublicPlan.Name))
				})

				It("is visible as having no access to an admin user", func() {
					workflowhelpers.AsUser(TestSetup.AdminUserContext(), TestSetup.ShortTimeout(), func() {
						commandResult := cf.Cf("disable-service-access", broker.Service.Name, "-p", orgPublicPlan.Name, "-o", TestSetup.RegularUserContext().Org).Wait(Config.DefaultTimeoutDuration())
						Expect(commandResult).To(Exit(0))
						commandResult = cf.Cf("disable-service-access", broker.Service.Name, "-p", globallyPublicPlan.Name).Wait(Config.DefaultTimeoutDuration())
						Expect(commandResult).To(Exit(0))
						acls = cf.Cf("service-access", "-e", broker.Service.Name).Wait(Config.DefaultTimeoutDuration())
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
			app_helpers.AppReport(broker.Name, Config.DefaultTimeoutDuration())

			broker.Destroy()
		})
	})

	Describe("private brokers", func() {
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
			app_helpers.AppReport(broker.Name, Config.DefaultTimeoutDuration())
			broker.Destroy()
		})

		It("can be created, viewed (in list), updated, and deleted by SpaceDevelopers", func() {
			workflowhelpers.AsUser(TestSetup.RegularUserContext(), TestSetup.ShortTimeout(), func() {
				spaceCmd := cf.Cf("space", TestSetup.RegularUserContext().Space, "--guid").Wait(Config.DefaultTimeoutDuration())
				spaceGuid := string(spaceCmd.Out.Contents())
				spaceGuid = strings.Trim(spaceGuid, "\n")
				body := map[string]string{
					"name":          broker.Name,
					"broker_url":    helpers.AppUri(broker.Name, "", Config),
					"auth_username": TestSetup.RegularUserContext().Username,
					"auth_password": TestSetup.RegularUserContext().Password,
					"space_guid":    spaceGuid,
				}
				jsonBody, _ := json.Marshal(body)

				By("Create")
				createBrokerCommand := cf.Cf("curl", "/v2/service_brokers", "-X", "POST", "-d", string(jsonBody)).Wait(Config.DefaultTimeoutDuration())
				Expect(createBrokerCommand).To(Exit(0))

				By("Read")
				serviceBrokersCommand := cf.Cf("service-brokers").Wait(Config.DefaultTimeoutDuration())
				Expect(serviceBrokersCommand).To(Exit(0))
				Expect(serviceBrokersCommand.Out.Contents()).To(ContainSubstring(broker.Name))

				By("Update")
				updatedName := broker.Name + "updated"
				updateBrokerCommand := cf.Cf("rename-service-broker", broker.Name, updatedName).Wait(Config.DefaultTimeoutDuration())
				Expect(updateBrokerCommand).To(Exit(0))

				serviceBrokersCommand = cf.Cf("service-brokers").Wait(Config.DefaultTimeoutDuration())
				Expect(serviceBrokersCommand).To(Exit(0))
				Expect(serviceBrokersCommand.Out.Contents()).To(ContainSubstring(updatedName))

				By("Delete")
				deleteBrokerCommand := cf.Cf("delete-service-broker", updatedName, "-f").Wait(Config.DefaultTimeoutDuration())
				Expect(deleteBrokerCommand).To(Exit(0))

				serviceBrokersCommand = cf.Cf("service-brokers").Wait(Config.DefaultTimeoutDuration())
				Expect(serviceBrokersCommand).To(Exit(0))
				Expect(serviceBrokersCommand.Out.Contents()).NotTo(ContainSubstring(updatedName))
			})
		})

		It("exposes the services and plans of the private broker in the space", func() {
			workflowhelpers.AsUser(TestSetup.RegularUserContext(), TestSetup.ShortTimeout(), func() {
				spaceCmd := cf.Cf("space", TestSetup.RegularUserContext().Space, "--guid").Wait(Config.DefaultTimeoutDuration())
				spaceGuid := string(spaceCmd.Out.Contents())
				spaceGuid = strings.Trim(spaceGuid, "\n")
				body := map[string]string{
					"name":          broker.Name,
					"broker_url":    helpers.AppUri(broker.Name, "", Config),
					"auth_username": TestSetup.RegularUserContext().Username,
					"auth_password": TestSetup.RegularUserContext().Password,
					"space_guid":    spaceGuid,
				}
				jsonBody, _ := json.Marshal(body)

				createBrokerCommand := cf.Cf("curl", "/v2/service_brokers", "-X", "POST", "-d", string(jsonBody)).Wait(Config.DefaultTimeoutDuration())
				Expect(createBrokerCommand).To(Exit(0))

				marketplaceOutput := cf.Cf("marketplace").Wait(Config.DefaultTimeoutDuration())

				Expect(marketplaceOutput).To(Exit(0))
				Expect(marketplaceOutput).To(Say(broker.Service.Name))
				Expect(marketplaceOutput).To(Say(broker.SyncPlans[0].Name))
				Expect(marketplaceOutput).To(Say(broker.SyncPlans[1].Name))
			})
		})
	})
})
