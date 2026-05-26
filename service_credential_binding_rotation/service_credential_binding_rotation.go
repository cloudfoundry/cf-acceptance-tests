package service_credential_binding_rotation

import (
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	svchelper "github.com/cloudfoundry/cf-acceptance-tests/helpers/services"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/helpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

type bindingResource struct {
	GUID      string    `json:"guid"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type bindingListResponse struct {
	Resources []bindingResource `json:"resources"`
}

type vcapServiceEntry struct {
	BindingName string `json:"binding_name"`
	BindingGUID string `json:"binding_guid"`
}

var _ = ServiceCredentialBindingRotationDescribe("Service Credential Binding Rotation", func() {
	Describe("rotation scenarios", Ordered, func() {
		var broker svchelper.ServiceBroker
		var appName string
		var serviceName string
		var bindingName string

		extractBindingGUIDFromVCAPServices := func(vcapServicesStr, serviceBindingName string) string {
			vcapServices := map[string][]vcapServiceEntry{}
			Expect(json.Unmarshal([]byte(vcapServicesStr), &vcapServices)).NotTo(HaveOccurred(), "failed to parse VCAP_SERVICES")

			serviceEntries := vcapServices[broker.Service.Name]
			for _, serviceEntry := range serviceEntries {
				if serviceEntry.BindingName == serviceBindingName {
					return serviceEntry.BindingGUID
				}
			}

			Fail(fmt.Sprintf("expected VCAP_SERVICES to contain binding_guid for service binding %q under label %q", serviceBindingName, broker.Service.Name))
			return ""
		}

		listBindingsForAppAndService := func(appName, serviceName string) []bindingResource {
			appGUID := app_helpers.GetAppGuid(appName)
			serviceGUID := svchelper.GetServiceInstanceGuid(serviceName)

			bindingEndpoint := fmt.Sprintf("/v3/service_credential_bindings?app_guids=%s&service_instance_guids=%s", appGUID, serviceGUID)
			session := cf.Cf("curl", bindingEndpoint).Wait()
			Expect(session).To(Exit(0), "failed to list service credential bindings")

			var response bindingListResponse
			Expect(json.Unmarshal(session.Out.Contents(), &response)).NotTo(HaveOccurred())
			return response.Resources
		}

		oldestBindingGUID := func(bindings []bindingResource) string {
			oldest := bindings[0]
			for _, binding := range bindings[1:] {
				if binding.CreatedAt.Before(oldest.CreatedAt) {
					oldest = binding
				}
			}
			return oldest.GUID
		}

		BeforeAll(func() {
			broker = svchelper.NewServiceBroker(
				random_name.CATSRandomName("BRKR"),
				assets.NewAssets().ServiceBroker,
				TestSetup,
			)
			broker.Push(Config)
			broker.Configure()
			broker.Create()
			broker.PublicizePlans()
		})

		AfterAll(func() {
			app_helpers.AppReport(broker.Name)
			broker.Destroy()
		})

		BeforeEach(func() {
			appName = random_name.CATSRandomName("APP")
			serviceName = random_name.CATSRandomName("SVIN")

			Expect(cf.Cf(app_helpers.CatnipWithArgs(
				appName,
				"-m", DEFAULT_MEMORY_LIMIT)...,
			).Wait(Config.CfPushTimeoutDuration())).To(Exit(0), "failed pushing app")

			Expect(cf.Cf("create-service", broker.Service.Name, broker.SyncPlans[0].Name, serviceName).Wait()).To(Exit(0))

			bindingName = random_name.CATSRandomName("BIND")
			Expect(cf.Cf("bind-service", appName, serviceName, "--binding-name", bindingName, "--strategy", "multiple").Wait()).To(
				Exit(0),
				fmt.Sprintf("failed binding app %s to service %s with binding name %s", appName, serviceName, bindingName),
			)

			Expect(cf.Cf("restage", appName, "--strategy", "rolling").Wait(Config.CfPushTimeoutDuration())).To(Exit(0), "failed rolling restage")
		})

		AfterEach(func() {
			app_helpers.AppReport(appName)

			Expect(cf.Cf("delete-service", serviceName, "-f").Wait()).To(Exit(0))
			Expect(cf.Cf("delete", appName, "-f", "-r").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		})

		Context("one binding exists for the test application and test service instance", func() {

			It("rotates credentials when creating the second binding", func() {
				vcapServices := helpers.CurlApp(Config, appName, "/env/VCAP_SERVICES")
				// test service broker supports only static credentials for service bindings,
				// so compare binding_guids to verify that the second bind-service call caused credential rotation (instead of being a no-op)
				initialBindingGUID := extractBindingGUIDFromVCAPServices(vcapServices, bindingName)

				secondBindSession := cf.Cf("bind-service", appName, serviceName, "--binding-name", bindingName, "--strategy", "multiple").Wait()
				Expect(secondBindSession).To(
					Exit(0),
					fmt.Sprintf("failed binding app %s to service %s with binding name %s", appName, serviceName, bindingName),
				)
				Expect(string(secondBindSession.Out.Contents())).ToNot(
					ContainSubstring(fmt.Sprintf("App %s is already bound to service instance %s.", appName, serviceName)),
					"Make sure to enable the multi-service-binding feature in your test backend.",
				)

				Expect(cf.Cf("restage", appName, "--strategy", "rolling").Wait(Config.CfPushTimeoutDuration())).To(Exit(0), "failed rolling restage")

				vcapServices = helpers.CurlApp(Config, appName, "/env/VCAP_SERVICES")
				rotatedBindingGUID := extractBindingGUIDFromVCAPServices(vcapServices, bindingName)

				Expect(rotatedBindingGUID).ToNot(Equal(initialBindingGUID), fmt.Sprintf("expected new service binding guid after completing credential rotation"))
			})
		})

		Context("two service bindings exist for the test application and test service instance", func() {

			BeforeEach(func() {
				secondBindSession := cf.Cf("bind-service", appName, serviceName, "--binding-name", bindingName, "--strategy", "multiple").Wait()
				Expect(secondBindSession).To(
					Exit(0),
					fmt.Sprintf("failed binding app %s to service %s with binding name %s", appName, serviceName, bindingName),
				)
				Expect(string(secondBindSession.Out.Contents())).ToNot(
					ContainSubstring(fmt.Sprintf("App %s is already bound to service instance %s.", appName, serviceName)),
					"Make sure to enable the multi-service-binding feature in your test backend.",
				)

				Expect(cf.Cf("restage", appName, "--strategy", "rolling").Wait(Config.CfPushTimeoutDuration())).To(Exit(0), "failed rolling restage")

				bindings := listBindingsForAppAndService(appName, serviceName)
				Expect(len(bindings)).To(Equal(2), fmt.Sprintf("expected two bindings for app %s and service %s", appName, serviceName))
			})

			Describe("show service instance information", func() {
				It("shows all bindings", func() {
					serviceSession := cf.Cf("service", serviceName).Wait()
					Expect(serviceSession).To(Exit(0))

					serviceOutput := string(serviceSession.Out.Contents())

					bindings := listBindingsForAppAndService(appName, serviceName)
					for _, binding := range bindings {
						/* "cf service SERVICE_INSTANCE" output has a table of bindings with columns like:
										name      binding name   status             message   guid                                   created_at
						                testapp                  create succeeded             9ec888c4-547c-4c7b-bc51-6f15a7821e5d   2026-03-16T12:37:31Z
						                testapp                  create succeeded             ccee98fb-8146-4a4f-8b72-a8f96d44f525   2026-03-16T12:16:27Z
						*/
						linePattern := fmt.Sprintf(
							`(?m)^\s+%s\s+%s\s+create succeeded\s+%s\s+\S+$`,
							regexp.QuoteMeta(appName),
							regexp.QuoteMeta(binding.Name),
							regexp.QuoteMeta(binding.GUID),
						)
						Expect(serviceOutput).To(
							MatchRegexp(linePattern),
							fmt.Sprintf("expected cf service output to contain row matching app=%s binding=%s guid=%s", appName, binding.Name, binding.GUID),
						)
					}
				})
			})

			Describe("unbind-service", func() {
				It("deletes both service bindings", func() {
					Expect(cf.Cf("unbind-service", appName, serviceName).Wait()).To(Exit(0))

					bindings := listBindingsForAppAndService(appName, serviceName)
					Expect(len(bindings)).To(Equal(0), fmt.Sprintf("expected no bindings for app %s and service %s after unbind-service", appName, serviceName))
				})
			})

			Describe("cleanup-outdated-service-bindings", func() {
				It("deletes the oldest binding", func() {
					bindings := listBindingsForAppAndService(appName, serviceName)
					oldestBindingGUID := oldestBindingGUID(bindings)

					Expect(cf.Cf("cleanup-outdated-service-bindings", appName, "--force").Wait()).To(Exit(0))

					bindings = listBindingsForAppAndService(appName, serviceName)
					Expect(len(bindings)).To(Equal(1), fmt.Sprintf("expected one binding for app %s and service %s after cleanup", appName, serviceName))
					Expect(bindings[0].GUID).ToNot(Equal(oldestBindingGUID), fmt.Sprintf("expected oldest binding for app %s / service %s to be deleted", appName, serviceName))
				})
			})
		})
	})
})
