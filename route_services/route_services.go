package route_services

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	logshelper "github.com/cloudfoundry/cf-acceptance-tests/helpers/logs"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = RouteServicesDescribe("Route Services", func() {
	Context("when a route binds to a service", func() {
		Context("when service broker returns a route service url", func() {
			var (
				serviceInstanceName      string
				brokerName               string
				appName                  string
				routeServiceName         string
				golangAsset              = assets.NewAssets().Golang
				loggingRouteServiceAsset = assets.NewAssets().LoggingRouteService
			)

			BeforeEach(func() {
				routeServiceName = random_name.CATSRandomName("APP")
				brokerName = random_name.CATSRandomName("BRKR")
				serviceInstanceName = random_name.CATSRandomName("SVIN")
				appName = random_name.CATSRandomName("APP")

				serviceName := random_name.CATSRandomName("SVC")
				brokerAppName := random_name.CATSRandomName("APP")

				createServiceBroker(brokerName, brokerAppName, serviceName)
				createServiceInstance(serviceInstanceName, serviceName)

				Expect(cf.Cf("push", appName,
					"-b", Config.GetGoBuildpackName(),
					"-m", DEFAULT_MEMORY_LIMIT,
					"-p", golangAsset,
					"-f", filepath.Join(golangAsset, "manifest.yml"),
				).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

					Expect(cf.Cf( "push", routeServiceName,
						"-b", Config.GetGoBuildpackName(),
						"-m", DEFAULT_MEMORY_LIMIT,
						"-p", loggingRouteServiceAsset,
						"-f", filepath.Join(loggingRouteServiceAsset, "manifest.yml"),
					).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

				configureBroker(brokerAppName, routeServiceName)
				bindRouteToService(appName, serviceInstanceName)
			})

			AfterEach(func() {
				app_helpers.AppReport(appName)
				app_helpers.AppReport(routeServiceName)

				unbindRouteFromService(appName, serviceInstanceName)
				deleteServiceInstance(serviceInstanceName)
				deleteServiceBroker(brokerName)

				Expect(cf.Cf("delete", appName, "-f", "-r").Wait()).To(Exit(0))
				Expect(cf.Cf("delete", routeServiceName, "-f", "-r").Wait()).To(Exit(0))
			})

			It("a request to the app is routed through the route service", func() {
				Eventually(func() *Session {
					helpers.CurlAppRoot(Config, appName)
					logs := logshelper.Tail(Config.GetUseLogCache(), routeServiceName)
					Expect(logs.Wait()).To(Exit(0))
					return logs
				}).Should(Say("Response Body: go, world"))
			})
		})

		Context("when service broker does not return a route service url", func() {
			var (
				serviceInstanceName string
				brokerName          string
				appName             string
				golangAsset         = assets.NewAssets().Golang
			)

			BeforeEach(func() {
				appName = random_name.CATSRandomName("APP")
				brokerName = random_name.CATSRandomName("BRKR")
				serviceInstanceName = random_name.CATSRandomName("SVIN")

				brokerAppName := random_name.CATSRandomName("APP")
				serviceName := random_name.CATSRandomName("SVC")

				createServiceBroker(brokerName, brokerAppName, serviceName)
				createServiceInstance(serviceInstanceName, serviceName)

				Expect(cf.Cf("push", appName,
					"-b", Config.GetGoBuildpackName(),
					"-m", DEFAULT_MEMORY_LIMIT,
					"-p", golangAsset,
					"-f", filepath.Join(golangAsset, "manifest.yml"),
				).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

				configureBroker(brokerAppName, "")

				bindRouteToService(appName, serviceInstanceName)
			})

			AfterEach(func() {
				app_helpers.AppReport(appName)

				unbindRouteFromService(appName, serviceInstanceName)
				deleteServiceInstance(serviceInstanceName)
				deleteServiceBroker(brokerName)

				Expect(cf.Cf("delete", appName, "-f", "-r").Wait()).To(Exit(0))
			})

			It("routes to an app", func() {
				Eventually(func() string {
					return helpers.CurlAppRoot(Config, appName)
				}).Should(ContainSubstring("go, world"))
			})
		})

		Context("when arbitrary parameters are sent", func() {
			var (
				serviceInstanceName string
				brokerName          string
				brokerAppName       string
				hostname            string
			)

			BeforeEach(func() {
				hostname = random_name.CATSRandomName("ROUTE")
				brokerAppName = random_name.CATSRandomName("APP")
				serviceInstanceName = random_name.CATSRandomName("SVIN")
				brokerName = random_name.CATSRandomName("BRKR")

				serviceName := random_name.CATSRandomName("SVC")

				createServiceBroker(brokerName, brokerAppName, serviceName)
				createServiceInstance(serviceInstanceName, serviceName)

				Expect(cf.Cf("create-route",
					TestSetup.RegularUserContext().Space,
					Config.GetAppsDomain(),
					"--hostname", hostname,
				).Wait()).To(Exit(0))

				configureBroker(brokerAppName, "")
			})

			AfterEach(func() {
				unbindRouteFromService(hostname, serviceInstanceName)
				deleteServiceInstance(serviceInstanceName)
				deleteServiceBroker(brokerName)

				Expect(cf.Cf("delete-route",
					Config.GetAppsDomain(),
					"--hostname", hostname,
					"-f",
				).Wait()).To(Exit(0))
			})

			It("passes them to the service broker", func() {
				bindRouteToServiceWithParams(hostname, serviceInstanceName, "{\"key1\":[\"value1\",\"irynaparam\"],\"key2\":\"value3\"}")

				Eventually(func() *Session {
					logs := logshelper.Tail(Config.GetUseLogCache(), brokerAppName)
					Expect(logs.Wait()).To(Exit(0))
					return logs
				}).Should(Say("irynaparam"))
			})
		})
	})
})

type customMap map[string]interface{}

func (c customMap) key(key string) customMap {
	return c[key].(map[string]interface{})
}

func bindRouteToService(hostname, serviceInstanceName string) {
	Expect(cf.Cf("bind-route-service", Config.GetAppsDomain(), serviceInstanceName,
		"-f",
		"--hostname", hostname,
	).Wait()).To(Exit(0))
}

func bindRouteToServiceWithParams(hostname, serviceInstanceName string, params string) {
	Expect(cf.Cf("bind-route-service", Config.GetAppsDomain(), serviceInstanceName,
		"-f",
		"--hostname", hostname,
		"-c", fmt.Sprintf("{\"parameters\": %s}", params),
	).Wait()).To(Exit(0))
}

func unbindRouteFromService(hostname, serviceInstanceName string) {
	Expect(cf.Cf("unbind-route-service", Config.GetAppsDomain(), serviceInstanceName,
		"-f",
		"--hostname", hostname,
	).Wait()).To(Exit(0))
}

func createServiceInstance(serviceInstanceName, serviceName string) {
	Expect(cf.Cf("create-service", serviceName, "fake-plan", serviceInstanceName).Wait()).To(Exit(0))
}

func deleteServiceInstance(serviceInstanceName string) {
	Expect(cf.Cf("delete-service", serviceInstanceName, "-f").Wait()).To(Exit(0))
}

func configureBroker(serviceBrokerAppName, routeServiceName string) {
	brokerConfigJson := helpers.CurlApp(Config, serviceBrokerAppName, "/config")

	var brokerConfigMap customMap

	err := json.Unmarshal([]byte(brokerConfigJson), &brokerConfigMap)
	Expect(err).NotTo(HaveOccurred())

	if routeServiceName != "" {
		routeServiceUrl := helpers.AppUri(routeServiceName, "/", Config)
		url, err := url.Parse(routeServiceUrl)
		Expect(err).NotTo(HaveOccurred())
		url.Scheme = "https"
		routeServiceUrl = url.String()

		brokerConfigMap.key("behaviors").key("bind").key("default").key("body")["route_service_url"] = routeServiceUrl
	} else {
		body := brokerConfigMap.key("behaviors").key("bind").key("default").key("body")
		delete(body, "route_service_url")
	}
	changedJson, err := json.Marshal(brokerConfigMap)
	Expect(err).NotTo(HaveOccurred())

	helpers.CurlApp(Config, serviceBrokerAppName, "/config", "-X", "POST", "-d", string(changedJson))
}

func createServiceBroker(brokerName, brokerAppName, serviceName string) {
	serviceBrokerAsset := assets.NewAssets().ServiceBroker
	Expect(cf.Cf("push", brokerAppName,
		"-b", Config.GetRubyBuildpackName(),
		"-m", DEFAULT_MEMORY_LIMIT,
		"-p", serviceBrokerAsset,
	).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

	initiateBrokerConfig(serviceName, brokerAppName)

	brokerUrl := helpers.AppUri(brokerAppName, "", Config)

	workflowhelpers.AsUser(TestSetup.AdminUserContext(), TestSetup.ShortTimeout(), func() {
		session := cf.Cf("create-service-broker", brokerName, "user", "password", brokerUrl)
		Expect(session.Wait()).To(Exit(0))

		session = cf.Cf("enable-service-access", serviceName)
		Expect(session.Wait()).To(Exit(0))
	})
}

func deleteServiceBroker(brokerName string) {
	workflowhelpers.AsUser(TestSetup.AdminUserContext(), TestSetup.ShortTimeout(), func() {
		responseBuffer := cf.Cf("delete-service-broker", brokerName, "-f")
		Expect(responseBuffer.Wait()).To(Exit(0))
	})
}

func initiateBrokerConfig(serviceName, serviceBrokerAppName string) {
	brokerConfigJson := helpers.CurlApp(Config, serviceBrokerAppName, "/config")

	var brokerConfigMap customMap

	err := json.Unmarshal([]byte(brokerConfigJson), &brokerConfigMap)
	Expect(err).NotTo(HaveOccurred())

	dashboardClientId := random_name.CATSRandomName("DASHBOARD-ID")
	serviceId := random_name.CATSRandomName("SVC-ID")

	services := brokerConfigMap.key("behaviors").key("catalog").key("body")["services"].([]interface{})
	service := services[0].(map[string]interface{})
	service["dashboard_client"].(map[string]interface{})["id"] = dashboardClientId
	service["name"] = serviceName
	service["id"] = serviceId

	plans := service["plans"].([]interface{})

	for i, plan := range plans {
		servicePlanId := random_name.CATSRandomName(fmt.Sprintf("SVC-PLAN-ID-%d-", i))
		plan.(map[string]interface{})["id"] = servicePlanId
	}

	changedJson, err := json.Marshal(brokerConfigMap)
	Expect(err).NotTo(HaveOccurred())

	helpers.CurlApp(Config, serviceBrokerAppName, "/config", "-X", "POST", "-d", string(changedJson))
}
