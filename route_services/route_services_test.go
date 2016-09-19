package route_services

import (
	"encoding/json"
	"fmt"
	"net/url"

	. "code.cloudfoundry.org/cf-routing-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/skip_messages"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = RouteServicesDescribe("Route Services", func() {
	BeforeEach(func() {
		if config.Backend != "diego" {
			Skip(skip_messages.SkipDiegoMessage)
		}
	})

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
				brokerName = random_name.CATSRandomName("BROKER")
				serviceInstanceName = random_name.CATSRandomName("SVCINS")
				appName = random_name.CATSRandomName("APP")

				serviceName := random_name.CATSRandomName("SVC")
				brokerAppName := random_name.CATSRandomName("APP")

				createServiceBroker(brokerName, brokerAppName, serviceName)
				createServiceInstance(serviceInstanceName, serviceName)

				PushAppNoStart(appName, golangAsset, config.GoBuildpackName, config.AppsDomain, CF_PUSH_TIMEOUT)
				EnableDiego(appName, DEFAULT_TIMEOUT)
				StartApp(appName, CF_PUSH_TIMEOUT)

				PushApp(routeServiceName, loggingRouteServiceAsset, config.GoBuildpackName, config.AppsDomain, CF_PUSH_TIMEOUT)
				configureBroker(brokerAppName, routeServiceName)

				bindRouteToService(appName, serviceInstanceName)
			})

			AfterEach(func() {
				AppReport(appName, DEFAULT_TIMEOUT)
				AppReport(routeServiceName, DEFAULT_TIMEOUT)

				unbindRouteFromService(appName, serviceInstanceName)
				deleteServiceInstance(serviceInstanceName)
				deleteServiceBroker(brokerName)
				DeleteApp(appName, DEFAULT_TIMEOUT)
				DeleteApp(routeServiceName, DEFAULT_TIMEOUT)
			})

			It("a request to the app is routed through the route service", func() {
				Eventually(func() *Session {
					helpers.CurlAppRoot(appName)
					logs := cf.Cf("logs", "--recent", routeServiceName)
					Expect(logs.Wait(DEFAULT_TIMEOUT)).To(Exit(0))
					return logs
				}, DEFAULT_TIMEOUT).Should(Say("Response Body: go, world"))
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
				brokerName = random_name.CATSRandomName("BROKER")
				serviceInstanceName = random_name.CATSRandomName("SVCINS")

				brokerAppName := random_name.CATSRandomName("APP")
				serviceName := random_name.CATSRandomName("SVC")

				createServiceBroker(brokerName, brokerAppName, serviceName)
				createServiceInstance(serviceInstanceName, serviceName)

				PushAppNoStart(appName, golangAsset, config.GoBuildpackName, config.AppsDomain, CF_PUSH_TIMEOUT)
				EnableDiego(appName, DEFAULT_TIMEOUT)
				StartApp(appName, CF_PUSH_TIMEOUT)

				configureBroker(brokerAppName, "")

				bindRouteToService(appName, serviceInstanceName)
			})

			AfterEach(func() {
				AppReport(appName, DEFAULT_TIMEOUT)

				unbindRouteFromService(appName, serviceInstanceName)
				deleteServiceInstance(serviceInstanceName)
				deleteServiceBroker(brokerName)
				DeleteApp(appName, DEFAULT_TIMEOUT)
			})

			It("routes to an app", func() {
				Eventually(func() string {
					return helpers.CurlAppRoot(appName)
				}, DEFAULT_TIMEOUT).Should(ContainSubstring("go, world"))
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
				serviceInstanceName = random_name.CATSRandomName("SVCINS")
				brokerName = random_name.CATSRandomName("BROKER")

				serviceName := random_name.CATSRandomName("SVC")

				createServiceBroker(brokerName, brokerAppName, serviceName)
				createServiceInstance(serviceInstanceName, serviceName)

				CreateRoute(hostname, "", testSetup.RegularUserContext().Space, config.AppsDomain, DEFAULT_TIMEOUT)

				configureBroker(brokerAppName, "")
			})

			AfterEach(func() {
				unbindRouteFromService(hostname, serviceInstanceName)
				deleteServiceInstance(serviceInstanceName)
				deleteServiceBroker(brokerName)
				DeleteRoute(hostname, "", config.AppsDomain, DEFAULT_TIMEOUT)
			})

			It("passes them to the service broker", func() {
				bindRouteToServiceWithParams(hostname, serviceInstanceName, "{\"key1\":[\"value1\",\"irynaparam\"],\"key2\":\"value3\"}")

				Eventually(func() *Session {
					logs := cf.Cf("logs", "--recent", brokerAppName)
					Expect(logs.Wait(DEFAULT_TIMEOUT)).To(Exit(0))
					return logs
				}, DEFAULT_TIMEOUT).Should(Say("irynaparam"))
			})
		})
	})
})

type customMap map[string]interface{}

func (c customMap) key(key string) customMap {
	return c[key].(map[string]interface{})
}

func bindRouteToService(hostname, serviceInstanceName string) {
	routeGuid := GetRouteGuid(hostname, "", DEFAULT_TIMEOUT)

	Expect(cf.Cf("bind-route-service", config.AppsDomain, serviceInstanceName,
		"-f",
		"--hostname", hostname,
	).Wait(DEFAULT_TIMEOUT)).To(Exit(0))

	Eventually(func() string {
		response := cf.Cf("curl", fmt.Sprintf("/v2/routes/%s", routeGuid))
		Expect(response.Wait(DEFAULT_TIMEOUT)).To(Exit(0))

		return string(response.Out.Contents())
	}, DEFAULT_TIMEOUT, "1s").ShouldNot(ContainSubstring(`"service_instance_guid": null`))
}

func bindRouteToServiceWithParams(hostname, serviceInstanceName string, params string) {
	routeGuid := GetRouteGuid(hostname, "", DEFAULT_TIMEOUT)
	Expect(cf.Cf("bind-route-service", config.AppsDomain, serviceInstanceName,
		"-f",
		"--hostname", hostname,
		"-c", fmt.Sprintf("{\"parameters\": %s}", params),
	).Wait(DEFAULT_TIMEOUT)).To(Exit(0))

	Eventually(func() string {
		response := cf.Cf("curl", fmt.Sprintf("/v2/routes/%s", routeGuid))
		Expect(response.Wait(DEFAULT_TIMEOUT)).To(Exit(0))

		return string(response.Out.Contents())
	}, DEFAULT_TIMEOUT, "1s").ShouldNot(ContainSubstring(`"service_instance_guid": null`))
}

func unbindRouteFromService(hostname, serviceInstanceName string) {
	routeGuid := GetRouteGuid(hostname, "", DEFAULT_TIMEOUT)
	Expect(cf.Cf("unbind-route-service", config.AppsDomain, serviceInstanceName,
		"-f",
		"--hostname", hostname,
	).Wait(DEFAULT_TIMEOUT)).To(Exit(0))

	Eventually(func() string {
		response := cf.Cf("curl", fmt.Sprintf("/v2/routes/%s", routeGuid))
		Expect(response.Wait(DEFAULT_TIMEOUT)).To(Exit(0))

		return string(response.Out.Contents())
	}, DEFAULT_TIMEOUT, "1s").Should(ContainSubstring(`"service_instance_guid": null`))
}

func createServiceInstance(serviceInstanceName, serviceName string) {
	Expect(cf.Cf("create-service", serviceName, "fake-plan", serviceInstanceName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
}

func deleteServiceInstance(serviceInstanceName string) {
	Expect(cf.Cf("delete-service", serviceInstanceName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
}

func configureBroker(serviceBrokerAppName, routeServiceName string) {
	brokerConfigJson := helpers.CurlApp(serviceBrokerAppName, "/config")

	var brokerConfigMap customMap

	err := json.Unmarshal([]byte(brokerConfigJson), &brokerConfigMap)
	Expect(err).NotTo(HaveOccurred())

	if routeServiceName != "" {
		routeServiceUrl := helpers.AppUri(routeServiceName, "/")
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

	helpers.CurlApp(serviceBrokerAppName, "/config", "-X", "POST", "-d", string(changedJson))
}

func createServiceBroker(brokerName, brokerAppName, serviceName string) {
	serviceBrokerAsset := assets.NewAssets().ServiceBroker
	PushApp(brokerAppName, serviceBrokerAsset, config.RubyBuildpackName, config.AppsDomain, CF_PUSH_TIMEOUT)

	initiateBrokerConfig(serviceName, brokerAppName)

	brokerUrl := helpers.AppUri(brokerAppName, "")

	testSetup := workflowhelpers.NewTestSuiteSetup(config)
	workflowhelpers.AsUser(testSetup.AdminUserContext(), testSetup.ShortTimeout(), func() {
		session := cf.Cf("create-service-broker", brokerName, "user", "password", brokerUrl)
		Expect(session.Wait(DEFAULT_TIMEOUT)).To(Exit(0))

		session = cf.Cf("enable-service-access", serviceName)
		Expect(session.Wait(DEFAULT_TIMEOUT)).To(Exit(0))

	})
}

func deleteServiceBroker(brokerName string) {
	testSetup := workflowhelpers.NewTestSuiteSetup(config)
	workflowhelpers.AsUser(testSetup.AdminUserContext(), testSetup.ShortTimeout(), func() {
		responseBuffer := cf.Cf("delete-service-broker", brokerName, "-f")
		Expect(responseBuffer.Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	})
}

func initiateBrokerConfig(serviceName, serviceBrokerAppName string) {
	brokerConfigJson := helpers.CurlApp(serviceBrokerAppName, "/config")

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

	helpers.CurlApp(serviceBrokerAppName, "/config", "-X", "POST", "-d", string(changedJson))
}
