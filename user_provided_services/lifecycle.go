package user_provided_services_test

import (
	"encoding/json"
	"fmt"
	"strings"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	"github.com/cloudfoundry/cf-test-helpers/cf"
	"github.com/cloudfoundry/cf-test-helpers/helpers"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

type LastOperation struct {
	State string `json:"state"`
}

type Resource struct {
	Name          string `json:"name"`
	GUID          string
	LastOperation LastOperation `json:"last_operation"`
}

type Response struct {
	Resources []Resource `json:"resources"`
}

type ErrorResponse struct {
	ErrorCode string `json:"error_code"`
}

var _ = UserProvidedServicesDescribe("Service Instance Lifecycle", func() {
	Context("service instances with no bindings", func() {
		var instanceName string
		AfterEach(func() {
			if instanceName != "" {
				Expect(cf.Cf("delete-service", instanceName, "-f").Wait()).To(Exit(0))
			}
		})

		It("can create a service instance", func() {
			tags := "['tag1', 'tag2']"
			creds := `{"param1": "value"}`

			instanceName = random_name.CATSRandomName("SVIN")
			createService := cf.Cf("create-user-provided-service", instanceName, "-p", creds, "-t", tags).Wait()
			Expect(createService).To(Exit(0))

			serviceInfo := cf.Cf("-v", "service", instanceName).Wait()
			Expect(serviceInfo).To(Say("(service|type):\\s+user-provided"))
			Expect(serviceInfo.Out.Contents()).To(MatchRegexp(`"tags":\s*\[\n.*tag1.*\n.*tag2.*\n.*\]`))
		})

		Context("when there is an existing service instance", func() {
			BeforeEach(func() {
				creds := `{"param1": "value"}`
				instanceName = random_name.CATSRandomName("SVIN")
				createService := cf.Cf("create-user-provided-service", instanceName, "-p", creds).Wait()
				Expect(createService).To(Exit(0))
			})

			It("fetch the credentials", func() {
				instanceGUID := getGuidFor("service", instanceName)
				credentials := cf.Cf("curl", fmt.Sprintf("/v3/service_instances/%s/credentials", instanceGUID)).Wait()
				Expect(credentials).To(Exit(0), "failed to curl fetch credentials")
				Expect(credentials).To(Say(`"param1": "value"`))
			})

			It("can delete a service instance", func() {
				deleteService := cf.Cf("delete-service", instanceName, "-f").Wait()
				Expect(deleteService).To(Exit(0))

				serviceInfo := cf.Cf("service", instanceName).Wait()
				combinedBuffer := BufferWithBytes(append(serviceInfo.Out.Contents(), serviceInfo.Err.Contents()...))
				Expect(combinedBuffer).To(Say("not found"))
			})

			Context("updating a service instance", func() {
				tags := "['tag1', 'tag2']"

				It("can rename a service", func() {
					newname := "newname"
					updateService := cf.Cf("rename-service", instanceName, newname).Wait()
					Expect(updateService).To(Exit(0))

					serviceInfo := cf.Cf("service", newname).Wait()
					Expect(serviceInfo).To(Say(newname))

					serviceInfo = cf.Cf("service", instanceName).Wait()
					Expect(serviceInfo).To(Exit(1))
				})

				It("can update service credentials", func() {
					newCreds := `{"param2": "newValue"}`
					updateService := cf.Cf("update-user-provided-service", instanceName, "-p", newCreds).Wait()
					Expect(updateService).To(Exit(0))

					instanceGUID := getGuidFor("service", instanceName)
					credentials := cf.Cf("curl", fmt.Sprintf("/v3/service_instances/%s/credentials", instanceGUID)).Wait()
					Expect(credentials).To(Exit(0), "failed to curl fetch credentials")
					Expect(credentials).To(Say(`"param2": "newValue"`))
				})

				It("can update service tags", func() {
					updateService := cf.Cf("update-user-provided-service", instanceName, "-t", tags).Wait()
					Expect(updateService).To(Exit(0))

					serviceInfo := cf.Cf("-v", "service", instanceName).Wait()
					Expect(serviceInfo.Out.Contents()).To(MatchRegexp(`"tags":\s*\[\n.*tag1.*\n.*tag2.*\n.*\]`))
				})
			})
		})
	})

	Context("service instances with bindings", func() {
		var instanceName, appName, username string

		BeforeEach(func() {
			appName = random_name.CATSRandomName("APP")
			createApp := cf.Cf(app_helpers.CatnipWithArgs(
				appName,
				"-m", DEFAULT_MEMORY_LIMIT)...,
			).Wait(Config.CfPushTimeoutDuration())
			Expect(createApp).To(Exit(0), "failed creating app")

			checkForAppEvent(appName, "audit.app.create")

			username = random_name.CATSRandomName("CREDENTIAL")
			creds := fmt.Sprintf(`{"username": "%s"}`, username)
			instanceName = random_name.CATSRandomName("SVIN")
			createService := cf.Cf("create-user-provided-service", instanceName, "-p", creds).Wait()
			Expect(createService).To(Exit(0), "failed creating service")
		})

		AfterEach(func() {
			app_helpers.AppReport(appName)
			Expect(cf.Cf("delete", appName, "-f", "-r").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
			Expect(cf.Cf("delete-service", instanceName, "-f").Wait()).To(Exit(0))
		})

		Describe("bindings", func() {
			It("can bind service to app and check app env and events", func() {
				bindService := cf.Cf("bind-service", appName, instanceName).Wait()
				Expect(bindService).To(Exit(0), "failed binding app to service")

				checkForAppEvent(appName, "audit.app.update")

				appEnv := cf.Cf("env", appName).Wait()
				Expect(appEnv).To(Exit(0), "failed get env for app")
				Expect(appEnv).To(Say("credentials"))

				restartApp := cf.Cf("restart", appName).Wait(Config.CfPushTimeoutDuration())
				Expect(restartApp).To(Exit(0), "failed restarting app")

				Expect(helpers.CurlApp(Config, appName, "/env/VCAP_SERVICES")).Should(ContainSubstring(username))
			})

			Context("when there is an existing binding", func() {
				BeforeEach(func() {
					bindService := cf.Cf("bind-service", appName, instanceName).Wait()
					Expect(bindService).To(Exit(0), "failed binding app to service")
				})

				It("can retrieve details", func() {
					appGUID := app_helpers.GetAppGuid(appName)
					serviceInstanceGUID := getGuidFor("service", instanceName)
					detailsEndpoint := getBindingDetailsEndpoint(appGUID, serviceInstanceGUID)

					fetchBindingDetails := cf.Cf("curl", detailsEndpoint).Wait()
					Expect(fetchBindingDetails).To(Say(`"username": "%s"`, username))
					Expect(fetchBindingDetails).To(Exit(0), "failed to fetch binding details")
				})

				It("updates to the service instance appear in app env", func() {
					restartApp := cf.Cf("restart", appName).Wait(Config.CfPushTimeoutDuration())
					Expect(restartApp).To(Exit(0), "failed restarting app")
					Expect(helpers.CurlApp(Config, appName, "/env/VCAP_SERVICES")).Should(ContainSubstring(username))

					newCreds := `{"username": "new-username"}`
					updateService := cf.Cf("update-user-provided-service", instanceName, "-p", newCreds).Wait()
					Expect(updateService).To(Exit(0))

					restartApp = cf.Cf("restart", appName).Wait(Config.CfPushTimeoutDuration())
					Expect(restartApp).To(Exit(0), "failed restarting app")
					Expect(helpers.CurlApp(Config, appName, "/env/VCAP_SERVICES")).Should(ContainSubstring("new-username"))
				})

				It("can unbind service to app and check app env and events", func() {
					unbindService := cf.Cf("unbind-service", appName, instanceName).Wait()
					Expect(unbindService).To(Exit(0), "failed unbinding app to service")

					checkForAppEvent(appName, "audit.app.update")

					appEnv := cf.Cf("env", appName).Wait()
					Expect(appEnv).To(Exit(0), "failed get env for app")
					Expect(appEnv).ToNot(Say("credentials"))

					restartApp := cf.Cf("restart", appName).Wait(Config.CfPushTimeoutDuration())
					Expect(restartApp).To(Exit(0), "failed restarting app")

					Expect(helpers.CurlApp(Config, appName, "/env/VCAP_SERVICES")).ShouldNot(ContainSubstring(username))
				})
			})
		})
	})
})

func checkForAppEvent(appName string, eventName string) {
	Eventually(func() string {
		return string(cf.Cf("events", appName).Wait().Out.Contents())
	}).Should(MatchRegexp(eventName))
}

func getBindingDetailsEndpoint(appGUID string, instanceGUID string) string {
	jsonResults := Response{}
	bindingCurl := cf.Cf("curl", fmt.Sprintf("/v3/service_credential_bindings?app_guids=%s&service_instance_guids=%s", appGUID, instanceGUID)).Wait()
	Expect(bindingCurl).To(Exit(0))
	Expect(json.Unmarshal(bindingCurl.Out.Contents(), &jsonResults)).NotTo(HaveOccurred())

	Expect(len(jsonResults.Resources)).To(BeNumerically(">", 0), "Expected to find at least one service resource.")

	return fmt.Sprintf("/v3/service_credential_bindings/%s/details", jsonResults.Resources[0].GUID)
}

func getGuidFor(args ...string) string {
	args = append(args, "--guid")
	session := cf.Cf(args...).Wait()

	out := string(session.Out.Contents())
	return strings.TrimSpace(out)
}
