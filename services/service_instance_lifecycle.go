package services_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/services"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

type LastOperation struct {
	State string `json:"state"`
}

type Entity struct {
	Name          string        `json:"name"`
	LastOperation LastOperation `json:"last_operation"`
}

type Metadata struct {
	URL  string
	GUID string
}

type Resource struct {
	Entity   Entity   `json:"entity"`
	Metadata Metadata `json:"metadata"`
}

type Response struct {
	Resources []Resource `json:"resources"`
}

type ErrorResponse struct {
	ErrorCode string `json:"error_code"`
}

var _ = ServicesDescribe("Service Instance Lifecycle", func() {
	var broker ServiceBroker
	var ASYNC_OPERATION_POLL_INTERVAL = 5 * time.Second

	waitForAsyncDeletionToComplete := func(broker ServiceBroker, instanceName string) {
		Eventually(func() *Buffer {
			session := cf.Cf("service", instanceName).Wait(Config.DefaultTimeoutDuration())
			combinedOutputBytes := append(session.Out.Contents(), session.Err.Contents()...)
			return BufferWithBytes(combinedOutputBytes)
		}, Config.AsyncServiceOperationTimeoutDuration(), ASYNC_OPERATION_POLL_INTERVAL).Should(Say("not found"))
	}

	waitForAsyncOperationToComplete := func(broker ServiceBroker, instanceName string) {
		Eventually(func() *Session {
			serviceDetails := cf.Cf("service", instanceName).Wait(Config.DefaultTimeoutDuration())
			Expect(serviceDetails).To(Exit(0), "failed getting service instance details")
			return serviceDetails
		}, Config.AsyncServiceOperationTimeoutDuration(), ASYNC_OPERATION_POLL_INTERVAL).Should(Say("succeeded"))
	}

	Describe("Synchronous operations", func() {
		BeforeEach(func() {
			broker = NewServiceBroker(
				random_name.CATSRandomName("BRKR"),
				assets.NewAssets().ServiceBroker,
				TestSetup,
			)
			broker.Push(Config)
			broker.Configure()
			broker.Create()
			broker.PublicizePlans()
		})

		AfterEach(func() {
			app_helpers.AppReport(broker.Name, Config.DefaultTimeoutDuration())

			broker.Destroy()
		})

		Describe("just service instances", func() {
			var instanceName string
			AfterEach(func() {
				if instanceName != "" {
					Expect(cf.Cf("delete-service", instanceName, "-f").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
				}
			})

			It("can create a service instance", func() {
				tags := "['tag1', 'tag2']"
				params := "{\"param1\": \"value\"}"

				instanceName = random_name.CATSRandomName("SVIN")
				createService := cf.Cf("create-service", broker.Service.Name, broker.SyncPlans[0].Name, instanceName, "-c", params, "-t", tags).Wait(Config.DefaultTimeoutDuration())
				Expect(createService).To(Exit(0))

				serviceInfo := cf.Cf("-v", "service", instanceName).Wait(Config.DefaultTimeoutDuration())
				Expect(serviceInfo).To(Say("[P|p]lan:\\s+%s", broker.SyncPlans[0].Name))
				Expect(serviceInfo.Out.Contents()).To(MatchRegexp(`"tags":\s*\[\n.*tag1.*\n.*tag2.*\n.*\]`))
			})

			Context("when there is an existing service instance", func() {
				BeforeEach(func() {
					params := "{\"param1\": \"value\"}"
					instanceName = random_name.CATSRandomName("SVIN")
					createService := cf.Cf("create-service", broker.Service.Name, broker.SyncPlans[0].Name, instanceName, "-c", params).Wait(Config.DefaultTimeoutDuration())
					Expect(createService).To(Exit(0), "failed creating service")
				})

				It("fetch the configuration parameters", func() {
					instanceGUID := getGuidFor("service", instanceName)
					configParams := cf.Cf("curl", fmt.Sprintf("/v2/service_instances/%s/parameters", instanceGUID)).Wait(Config.DefaultTimeoutDuration())
					Expect(configParams).To(Exit(0), "failed to curl fetch binding parameters")
					Expect(configParams).To(Say("\"param1\": \"value\""))
				})

				It("can delete a service instance", func() {
					deleteService := cf.Cf("delete-service", instanceName, "-f").Wait(Config.DefaultTimeoutDuration())
					Expect(deleteService).To(Exit(0))

					serviceInfo := cf.Cf("service", instanceName).Wait(Config.DefaultTimeoutDuration())
					combinedBuffer := BufferWithBytes(append(serviceInfo.Out.Contents(), serviceInfo.Err.Contents()...))
					Expect(combinedBuffer).To(Say("not found"))
				})

				Context("updating a service instance", func() {
					tags := "['tag1', 'tag2']"
					params := "{\"param1\": \"value\"}"

					It("can rename a service", func() {
						newname := "newname"
						updateService := cf.Cf("rename-service", instanceName, newname).Wait(Config.DefaultTimeoutDuration())
						Expect(updateService).To(Exit(0))

						serviceInfo := cf.Cf("service", newname).Wait(Config.DefaultTimeoutDuration())
						Expect(serviceInfo).To(Say(newname))

						serviceInfo = cf.Cf("service", instanceName).Wait(Config.DefaultTimeoutDuration())
						Expect(serviceInfo).To(Exit(1))
					})

					It("can update a service plan", func() {
						updateService := cf.Cf("update-service", instanceName, "-p", broker.SyncPlans[1].Name).Wait(Config.DefaultTimeoutDuration())
						Expect(updateService).To(Exit(0))

						serviceInfo := cf.Cf("service", instanceName).Wait(Config.DefaultTimeoutDuration())
						Expect(serviceInfo).To(Say("[P|p]lan:\\s+%s", broker.SyncPlans[1].Name))
					})

					It("can update service tags", func() {
						updateService := cf.Cf("update-service", instanceName, "-t", tags).Wait(Config.DefaultTimeoutDuration())
						Expect(updateService).To(Exit(0))

						serviceInfo := cf.Cf("-v", "service", instanceName).Wait(Config.DefaultTimeoutDuration())
						Expect(serviceInfo.Out.Contents()).To(MatchRegexp(`"tags":\s*\[\n.*tag1.*\n.*tag2.*\n.*\]`))
					})

					It("can update arbitrary parameters", func() {
						updateService := cf.Cf("update-service", instanceName, "-c", params).Wait(Config.DefaultTimeoutDuration())
						Expect(updateService).To(Exit(0), "Failed updating service")
						//Note: We don't necessarily get these back through a service instance lookup
					})

					It("can update all available parameters at once", func() {
						updateService := cf.Cf(
							"update-service", instanceName,
							"-p", broker.SyncPlans[1].Name,
							"-t", tags,
							"-c", params).Wait(Config.DefaultTimeoutDuration())
						Expect(updateService).To(Exit(0))

						serviceInfo := cf.Cf("-v", "service", instanceName).Wait(Config.DefaultTimeoutDuration())
						Expect(serviceInfo).To(Say("[P|p]lan:\\s+%s", broker.SyncPlans[1].Name))
						Expect(serviceInfo.Out.Contents()).To(MatchRegexp(`"tags":\s*\[\n.*tag1.*\n.*tag2.*\n.*\]`))
					})
				})

				Describe("service keys", func() {
					var keyName string
					BeforeEach(func() {
						keyName = random_name.CATSRandomName("SVC-KEY")
					})

					AfterEach(func() {
						Expect(cf.Cf("delete-service-key", instanceName, keyName, "-f").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
					})

					It("can create service keys", func() {
						createKey := cf.Cf("create-service-key", instanceName, keyName).Wait(Config.DefaultTimeoutDuration())
						Expect(createKey).To(Exit(0), "failed to create key")

						keyInfo := cf.Cf("service-key", instanceName, keyName).Wait(Config.DefaultTimeoutDuration())
						Expect(keyInfo).To(Exit(0), "failed key info")

						Expect(keyInfo).To(Say(`"database": "fake-dbname"`))
						Expect(keyInfo).To(Say(`"password": "fake-password"`))
						Expect(keyInfo).To(Say(`"username": "fake-user"`))
					})

					It("can create service keys with arbitrary params", func() {
						params := "{\"param1\": \"value\"}"
						createKey := cf.Cf("create-service-key", instanceName, keyName, "-c", params).Wait(Config.DefaultTimeoutDuration())
						Expect(createKey).To(Exit(0), "failed creating key with params")
					})

					Context("when there is an existing key", func() {
						BeforeEach(func() {
							params := "{\"param1\": \"value\"}"
							createKey := cf.Cf("create-service-key", instanceName, keyName, "-c", params).Wait(Config.DefaultTimeoutDuration())
							Expect(createKey).To(Exit(0), "failed to create key")
						})

						It("can retrieve parameters", func() {
							serviceKeyGUID := getGuidFor("service-key", instanceName, keyName)
							paramsEndpoint := fmt.Sprintf("/v2/service_keys/%s/parameters", serviceKeyGUID)

							fetchServiceKeyParameters := cf.Cf("curl", paramsEndpoint).Wait(Config.DefaultTimeoutDuration())
							Expect(fetchServiceKeyParameters).To(Say(`"param1": "value"`))
							Expect(fetchServiceKeyParameters).To(Exit(0), "failed to fetch service key parameters")
						})

						It("can delete the key", func() {
							deleteServiceKey := cf.Cf("delete-service-key", instanceName, keyName, "-f").Wait(Config.DefaultTimeoutDuration())
							Expect(deleteServiceKey).To(Exit(0), "failed deleting service key")

							keyInfo := cf.Cf("service-key", instanceName, keyName).Wait(Config.DefaultTimeoutDuration())
							Expect(keyInfo).To(Say(fmt.Sprintf("No service key %s found for service instance %s", keyName, instanceName)))
						})
					})
				})
			})
		})

		Context("when there is an app", func() {
			var instanceName, appName string

			BeforeEach(func() {
				appName = random_name.CATSRandomName("APP")
				createApp := cf.Cf("push",
					appName,
					"-b", Config.GetBinaryBuildpackName(),
					"-m", DEFAULT_MEMORY_LIMIT,
					"-p", assets.NewAssets().Catnip,
					"-c", "./catnip",
					"-d", Config.GetAppsDomain()).Wait(Config.CfPushTimeoutDuration())
				Expect(createApp).To(Exit(0), "failed creating app")

				checkForAppEvents(appName, []string{"audit.app.create"})

				instanceName = random_name.CATSRandomName("SVIN")
				createService := cf.Cf("create-service", broker.Service.Name, broker.SyncPlans[0].Name, instanceName).Wait(Config.DefaultTimeoutDuration())
				Expect(createService).To(Exit(0), "failed creating service")
			})

			AfterEach(func() {
				app_helpers.AppReport(appName, Config.DefaultTimeoutDuration())
				Expect(cf.Cf("delete", appName, "-f", "-r").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
				Expect(cf.Cf("delete-service", instanceName, "-f").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
			})

			Describe("bindings", func() {
				It("can bind service to app and check app env and events", func() {
					bindService := cf.Cf("bind-service", appName, instanceName).Wait(Config.DefaultTimeoutDuration())
					Expect(bindService).To(Exit(0), "failed binding app to service")

					checkForAppEvents(appName, []string{"audit.app.update"})

					appEnv := cf.Cf("env", appName).Wait(Config.DefaultTimeoutDuration())
					Expect(appEnv).To(Exit(0), "failed get env for app")
					Expect(appEnv).To(Say(fmt.Sprintf("credentials")))

					restartApp := cf.Cf("restart", appName).Wait(Config.CfPushTimeoutDuration())
					Expect(restartApp).To(Exit(0), "failed restarting app")

					Expect(helpers.CurlApp(Config, appName, "/env/VCAP_SERVICES")).Should(ContainSubstring("fake-service://fake-user:fake-password@fake-host:3306/fake-dbname"))
				})

				It("can bind service to app and send arbitrary params", func() {
					bindService := cf.Cf("bind-service", appName, instanceName, "-c", `{"param1": "value"}`).Wait(Config.DefaultTimeoutDuration())
					Expect(bindService).To(Exit(0), "failed binding app to service")
				})

				Context("when there is an existing binding", func() {
					BeforeEach(func() {
						bindService := cf.Cf("bind-service", appName, instanceName, "-c", `{"max_clients": 5}`).Wait(Config.DefaultTimeoutDuration())
						Expect(bindService).To(Exit(0), "failed binding app to service")
					})

					It("can retrieve parameters", func() {
						appGUID := app_helpers.GetAppGuid(appName)
						serviceInstanceGUID := getGuidFor("service", instanceName)
						paramsEndpoint := getBindingParamsEndpoint(appGUID, serviceInstanceGUID)

						fetchBindingParameters := cf.Cf("curl", paramsEndpoint).Wait(Config.DefaultTimeoutDuration())
						Expect(fetchBindingParameters).To(Say(`"max_clients": 5`))
						Expect(fetchBindingParameters).To(Exit(0), "failed to fetch binding parameters")
					})

					It("can unbind service to app and check app env and events", func() {
						unbindService := cf.Cf("unbind-service", appName, instanceName).Wait(Config.DefaultTimeoutDuration())
						Expect(unbindService).To(Exit(0), "failed unbinding app to service")

						checkForAppEvents(appName, []string{"audit.app.update"})

						appEnv := cf.Cf("env", appName).Wait(Config.DefaultTimeoutDuration())
						Expect(appEnv).To(Exit(0), "failed get env for app")
						Expect(appEnv).ToNot(Say(fmt.Sprintf("credentials")))

						restartApp := cf.Cf("restart", appName).Wait(Config.CfPushTimeoutDuration())
						Expect(restartApp).To(Exit(0), "failed restarting app")

						Expect(helpers.CurlApp(Config, appName, "/env/VCAP_SERVICES")).ShouldNot(ContainSubstring("fake-service://fake-user:fake-password@fake-host:3306/fake-dbname"))
					})
				})
			})
		})
	})

	Describe("Asynchronous operations", func() {
		var instanceName string

		BeforeEach(func() {
			broker = NewServiceBroker(
				random_name.CATSRandomName("BRKR"),
				assets.NewAssets().ServiceBroker,
				TestSetup,
			)
			broker.Push(Config)
			broker.Configure()
			broker.Create()
			broker.PublicizePlans()
		})

		AfterEach(func() {
			app_helpers.AppReport(broker.Name, Config.DefaultTimeoutDuration())

			Expect(cf.Cf("delete-service", instanceName, "-f").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
			waitForAsyncDeletionToComplete(broker, instanceName)

			broker.Destroy()
		})

		Describe("for a service instance", func() {
			It("can create a service instance", func() {
				tags := "['tag1', 'tag2']"
				params := "{\"param1\": \"value\"}"

				instanceName = random_name.CATSRandomName("SVIN")
				createService := cf.Cf("create-service", broker.Service.Name, broker.AsyncPlans[0].Name, instanceName, "-t", tags, "-c", params).Wait(Config.DefaultTimeoutDuration())
				Expect(createService).To(Exit(0))
				Expect(createService).To(Say("Create in progress."))

				waitForAsyncOperationToComplete(broker, instanceName)

				serviceInfo := cf.Cf("-v", "service", instanceName).Wait(Config.DefaultTimeoutDuration())
				Expect(serviceInfo).To(Say("[P|p]lan:\\s+%s", broker.AsyncPlans[0].Name))
				Expect(serviceInfo).To(Say("[S|s]tatus:\\s+create succeeded"))
				Expect(serviceInfo).To(Say("[M|m]essage:\\s+100 percent done"))
				Expect(serviceInfo.Out.Contents()).To(MatchRegexp(`"tags":\s*\[\n.*tag1.*\n.*tag2.*\n.*\]`))
			})

			Context("when there is an existing service instance", func() {
				tags := "['tag1', 'tag2']"
				params := "{\"param1\": \"value2\"}"

				BeforeEach(func() {
					instanceName = random_name.CATSRandomName("SVC")
					createService := cf.Cf("create-service", broker.Service.Name, broker.AsyncPlans[0].Name, instanceName).Wait(Config.DefaultTimeoutDuration())
					Expect(createService).To(Exit(0))
					Expect(createService).To(Say("Create in progress."))

					waitForAsyncOperationToComplete(broker, instanceName)
				})

				It("can update a service plan", func() {
					updateService := cf.Cf("update-service", instanceName, "-p", broker.AsyncPlans[1].Name).Wait(Config.DefaultTimeoutDuration())
					Expect(updateService).To(Exit(0))
					Expect(updateService).To(Say("Update in progress."))

					serviceInfo := cf.Cf("service", instanceName).Wait(Config.DefaultTimeoutDuration())
					Expect(serviceInfo).To(Exit(0), "failed getting service instance details")
					Expect(serviceInfo).To(Say("[P|p]lan:\\s+%s", broker.AsyncPlans[0].Name))

					waitForAsyncOperationToComplete(broker, instanceName)

					serviceInfo = cf.Cf("service", instanceName).Wait(Config.DefaultTimeoutDuration())
					Expect(serviceInfo).To(Exit(0), "failed getting service instance details")
					Expect(serviceInfo).To(Say("[P|p]lan:\\s+%s", broker.AsyncPlans[1].Name))
				})

				It("can update the arbitrary params", func() {
					updateService := cf.Cf("update-service", instanceName, "-c", params).Wait(Config.DefaultTimeoutDuration())
					Expect(updateService).To(Exit(0))
					Expect(updateService).To(Say("Update in progress."))

					waitForAsyncOperationToComplete(broker, instanceName)
				})

				It("can update all of the possible parameters at once", func() {
					updateService := cf.Cf(
						"update-service", instanceName,
						"-t", tags,
						"-c", params,
						"-p", broker.AsyncPlans[1].Name).Wait(Config.DefaultTimeoutDuration())
					Expect(updateService).To(Exit(0))
					Expect(updateService).To(Say("Update in progress."))

					waitForAsyncOperationToComplete(broker, instanceName)

					serviceInfo := cf.Cf("-v", "service", instanceName).Wait(Config.DefaultTimeoutDuration())
					Expect(serviceInfo).To(Exit(0), "failed getting service instance details")
					Expect(serviceInfo).To(Say("[P|p]lan:\\s+%s", broker.AsyncPlans[1].Name))
					Expect(serviceInfo.Out.Contents()).To(MatchRegexp(`"tags":\s*\[\n.*tag1.*\n.*tag2.*\n.*\]`))
				})

				It("can delete a service instance", func() {
					deleteService := cf.Cf("delete-service", instanceName, "-f").Wait(Config.DefaultTimeoutDuration())
					Expect(deleteService).To(Exit(0), "failed making delete request")
					Expect(deleteService).To(Say("Delete in progress."))

					waitForAsyncDeletionToComplete(broker, instanceName)
				})

				Context("when there is an app", func() {
					var appName string
					BeforeEach(func() {
						appName = random_name.CATSRandomName("APP")
						createApp := cf.Cf("push",
							appName,
							"-b", Config.GetBinaryBuildpackName(),
							"-m", DEFAULT_MEMORY_LIMIT,
							"-p", assets.NewAssets().Catnip,
							"-c", "./catnip",
							"-d", Config.GetAppsDomain()).Wait(Config.CfPushTimeoutDuration())
						Expect(createApp).To(Exit(0), "failed creating app")
					})

					AfterEach(func() {
						app_helpers.AppReport(appName, Config.DefaultTimeoutDuration())
						Expect(cf.Cf("delete", appName, "-f", "-r").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
					})

					It("can bind a service instance", func() {
						bindService := cf.Cf("bind-service", appName, instanceName).Wait(Config.DefaultTimeoutDuration())
						Expect(bindService).To(Exit(0), "failed binding app to service")

						checkForAppEvents(appName, []string{"audit.app.update"})

						restageApp := cf.Cf("restage", appName).Wait(Config.CfPushTimeoutDuration())
						Expect(restageApp).To(Exit(0), "failed restaging app")

						checkForAppEvents(appName, []string{"audit.app.restage"})

						appEnv := cf.Cf("env", appName).Wait(Config.DefaultTimeoutDuration())
						Expect(appEnv).To(Exit(0), "failed get env for app")
						Expect(appEnv).To(Say(fmt.Sprintf("credentials")))
					})

					It("can bind service to app and send arbitrary params", func() {
						bindService := cf.Cf("bind-service", appName, instanceName, "-c", params).Wait(Config.DefaultTimeoutDuration())
						Expect(bindService).To(Exit(0), "failed binding app to service")

						checkForAppEvents(appName, []string{"audit.app.update"})
					})

					Context("when there is an existing binding", func() {
						BeforeEach(func() {
							bindService := cf.Cf("bind-service", appName, instanceName).Wait(Config.DefaultTimeoutDuration())
							Expect(bindService).To(Exit(0), "failed binding app to service")
						})

						It("can unbind a service instance", func() {
							unbindService := cf.Cf("unbind-service", appName, instanceName).Wait(Config.DefaultTimeoutDuration())
							Expect(unbindService).To(Exit(0), "failed unbinding app to service")

							checkForAppEvents(appName, []string{"audit.app.update"})

							appEnv := cf.Cf("env", appName).Wait(Config.DefaultTimeoutDuration())
							Expect(appEnv).To(Exit(0), "failed get env for app")
							Expect(appEnv).ToNot(Say(fmt.Sprintf("credentials")))
						})
					})
				})
			})
		})

		Describe("for a service binding", func() {
			var (
				appName     string
				appGUID     string
				serviceGUID string
			)

			BeforeEach(func() {
				instanceName = random_name.CATSRandomName("SVC")
				createService := cf.Cf("create-service", broker.Service.Name, broker.AsyncPlans[2].Name, instanceName).Wait(Config.DefaultTimeoutDuration())
				Expect(createService).To(Exit(0))
				Expect(createService).To(Say("Create in progress."))

				waitForAsyncOperationToComplete(broker, instanceName)

				appName = random_name.CATSRandomName("APP")
				createApp := cf.Cf("push",
					appName,
					"--no-start",
					"-b", Config.GetBinaryBuildpackName(),
					"-m", DEFAULT_MEMORY_LIMIT,
					"-p", assets.NewAssets().Catnip,
					"-c", "./catnip",
					"-d", Config.GetAppsDomain()).Wait(Config.DefaultTimeoutDuration())
				Expect(createApp).To(Exit(0), "failed creating app")
				Expect(cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

				appGUID = getGuidFor("app", appName)
				serviceGUID = getGuidFor("service", instanceName)
			})

			AfterEach(func() {
				app_helpers.AppReport(appName, Config.DefaultTimeoutDuration())
				Expect(cf.Cf("delete", appName, "-f", "-r").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
			})

			It("can bind and unbind asynchronously", func() {
				By("creating a binding asynchronously")
				bindService := cf.Cf(
					"curl", "/v2/service_bindings?accepts_incomplete=true", "-X", "POST",
					"-d", fmt.Sprintf(`'{ "app_guid": "%s", "service_instance_guid": "%s" }'`, appGUID, serviceGUID)).
					Wait(Config.DefaultTimeoutDuration())

				Expect(bindService).To(Exit(0), "failed to asynchronously bind service")

				var bindingResource Resource
				err := json.Unmarshal(bindService.Out.Contents(), &bindingResource)
				Expect(err).NotTo(HaveOccurred())
				bindingMetadata := bindingResource.Metadata

				By("waiting for binding to be created")
				Eventually(func() string {
					bindingDetails := cf.Cf("curl", bindingMetadata.URL).Wait(Config.DefaultTimeoutDuration())
					Expect(bindingDetails).To(Exit(0), "failed getting service binding details")

					var binding Resource
					err := json.Unmarshal(bindingDetails.Out.Contents(), &binding)
					Expect(err).NotTo(HaveOccurred())

					return binding.Entity.LastOperation.State
				}, Config.AsyncServiceOperationTimeoutDuration(), ASYNC_OPERATION_POLL_INTERVAL).Should(Equal("succeeded"))

				appEnv := cf.Cf("env", appName).Wait(Config.DefaultTimeoutDuration())
				Expect(appEnv).To(Exit(0), "failed get env for app")
				Expect(appEnv).To(Say(fmt.Sprintf("credentials")))

				restartApp := cf.Cf("restart", appName).Wait(Config.CfPushTimeoutDuration())
				Expect(restartApp).To(Exit(0), "failed restarting app")

				Expect(helpers.CurlApp(Config, appName, "/env/VCAP_SERVICES")).Should(ContainSubstring("fake-service://fake-user:fake-password@fake-host:3306/fake-dbname"))
			})
		})
	})
})

func checkForAppEvents(name string, eventNames []string) {
	events := cf.Cf("events", name).Wait(Config.DefaultTimeoutDuration())
	Expect(events).To(Exit(0), fmt.Sprintf("failed getting events for %s", name))

	for _, eventName := range eventNames {
		Expect(events).To(Say(eventName), "failed to find event")
	}
}

func getBindingParamsEndpoint(appGUID string, instanceGUID string) string {
	jsonResults := Response{}
	bindingCurl := cf.Cf("curl", fmt.Sprintf("/v2/apps/%s/service_bindings?q=service_instance_guid:%s", appGUID, instanceGUID)).Wait(Config.DefaultTimeoutDuration())
	Expect(bindingCurl).To(Exit(0))
	json.Unmarshal(bindingCurl.Out.Contents(), &jsonResults)

	Expect(len(jsonResults.Resources)).To(BeNumerically(">", 0), "Expected to find at least one service resource.")

	return fmt.Sprintf("%s/parameters", jsonResults.Resources[0].Metadata.URL)
}

func getGuidFor(args ...string) string {
	args = append(args, "--guid")
	session := cf.Cf(args...).Wait(Config.DefaultTimeoutDuration())

	out := string(session.Out.Contents())
	return strings.TrimSpace(out)
}
