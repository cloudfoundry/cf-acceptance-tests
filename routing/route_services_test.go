package routing

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Route Services", func() {
	config := helpers.LoadConfig()
	if config.IncludeRouteServices {
		Context("when a route binds to a service", func() {
			Context("when service broker returns a route service url", func() {
				var (
					brokerName               string
					brokerAppName            string
					serviceInstanceName      string
					appName                  string
					routeServiceName         string
					golangAsset              = assets.NewAssets().Golang
					loggingRouteServiceAsset = assets.NewAssets().LoggingRouteServiceZip
				)

				BeforeEach(func() {
					var serviceName string
					brokerName, brokerAppName, serviceName = createServiceBroker()
					serviceInstanceName = createServiceInstance(serviceName)

					appName = PushApp(golangAsset, config.GoBuildpackName)
					EnableDiego(appName)

					routeServiceName = PushApp(loggingRouteServiceAsset, config.GoBuildpackName)
					configureBroker(brokerAppName, routeServiceName)

					bindRouteToService(appName, serviceInstanceName)
				})

				AfterEach(func() {
					app_helpers.AppReport(appName, DEFAULT_TIMEOUT)

					unbindRouteFromService(appName, serviceInstanceName)
					deleteServiceInstance(serviceInstanceName)
					deleteServiceBroker(brokerName)
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
					brokerName          string
					brokerAppName       string
					serviceInstanceName string
					appName             string
					golangAsset         = assets.NewAssets().Golang
				)

				BeforeEach(func() {
					var serviceName string
					brokerName, brokerAppName, serviceName = createServiceBroker()
					serviceInstanceName = createServiceInstance(serviceName)
					appName = PushApp(golangAsset, config.GoBuildpackName)
					EnableDiego(appName)

					configureBroker(brokerAppName, "")

					bindRouteToService(appName, serviceInstanceName)
				})

				AfterEach(func() {
					app_helpers.AppReport(appName, DEFAULT_TIMEOUT)

					unbindRouteFromService(appName, serviceInstanceName)
					deleteServiceInstance(serviceInstanceName)
					deleteServiceBroker(brokerName)
				})

				It("routes to an app", func() {
					Eventually(func() string {
						return helpers.CurlAppRoot(appName)
					}, DEFAULT_TIMEOUT).Should(ContainSubstring("go, world"))
				})
			})
		})
	}
})

type customMap map[string]interface{}

func (c customMap) key(key string) customMap {
	return c[key].(map[string]interface{})
}

func bindRouteToService(routeName, serviceInstanceName string) {
	routeGuid := getRouteGuid(routeName)
	serviceInstanceGuid := getServiceInstanceGuid(serviceInstanceName)
	cf.Cf("curl", fmt.Sprintf("/v2/service_instances/%s/routes/%s", serviceInstanceGuid, routeGuid), "-X", "PUT")

	Eventually(func() string {
		response := cf.Cf("curl", fmt.Sprintf("/v2/routes/%s", routeGuid))
		Expect(response.Wait(DEFAULT_TIMEOUT)).To(Exit(0))

		contents := response.Out.Contents()
		return string(contents)
	}, DEFAULT_TIMEOUT, "1s").ShouldNot(ContainSubstring(`"service_instance_guid": null`))
}

func deleteServiceBroker(brokerName string) {
	config = helpers.LoadConfig()
	context := helpers.NewContext(config)
	cf.AsUser(context.AdminUserContext(), context.ShortTimeout(), func() {
		responseBuffer := cf.Cf("delete-service-broker", brokerName, "-f")
		Expect(responseBuffer.Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	})
}

func deleteServiceInstance(serviceInstanceName string) {
	guid := getServiceInstanceGuid(serviceInstanceName)
	Eventually(func() string {
		response := cf.Cf("curl", fmt.Sprintf("/v2/service_instances/%s", guid), "-X", "DELETE")
		Expect(response.Wait(DEFAULT_TIMEOUT)).To(Exit(0))

		contents := response.Out.Contents()
		return string(contents)
	}, DEFAULT_TIMEOUT, "1s").Should(ContainSubstring("CF-ServiceInstanceNotFound"))
}

func unbindRouteFromService(routeName, serviceInstanceName string) {
	route_guid := getRouteGuid(routeName)
	guid := getServiceInstanceGuid(serviceInstanceName)
	cf.Cf("curl", fmt.Sprintf("/v2/service_instances/%s/routes/%s", guid, route_guid), "-X", "DELETE")

	Eventually(func() string {
		response := cf.Cf("curl", fmt.Sprintf("/v2/routes/%s", route_guid))
		Expect(response.Wait(DEFAULT_TIMEOUT)).To(Exit(0))

		contents := response.Out.Contents()
		return string(contents)
	}, DEFAULT_TIMEOUT, "1s").Should(ContainSubstring(`"service_instance_guid": null`))
}

func getRouteGuid(hostname string) string {
	responseBuffer := cf.Cf("curl", fmt.Sprintf("/v2/routes?q=host:%s", hostname))
	Expect(responseBuffer.Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	routeBytes := responseBuffer.Out.Contents()

	var routeMap response

	err := json.Unmarshal(routeBytes, &routeMap)
	Expect(err).NotTo(HaveOccurred())

	return routeMap.Resources[0].Metadata.Guid
}

type metadata struct {
	Guid string
}
type resource struct {
	Metadata metadata
}
type response struct {
	Resources []resource
}

func getServiceInstanceGuid(serviceInstanceName string) string {
	serviceInstanceBuffer := cf.Cf("curl", fmt.Sprintf("/v2/service_instances?q=name:%s", serviceInstanceName))
	Expect(serviceInstanceBuffer.Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	serviceInstanceBytes := serviceInstanceBuffer.Out.Contents()

	var serviceInstanceMap response

	err := json.Unmarshal(serviceInstanceBytes, &serviceInstanceMap)
	Expect(err).NotTo(HaveOccurred())

	return serviceInstanceMap.Resources[0].Metadata.Guid
}

func createServiceInstance(serviceName string) string {
	serviceInstanceName := generator.PrefixedRandomName("RATS-SERVICE-")

	session := cf.Cf("create-service", serviceName, "fake-plan", serviceInstanceName)
	Expect(session.Wait(DEFAULT_TIMEOUT)).To(Exit(0))

	return serviceInstanceName
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

func createServiceBroker() (string, string, string) {
	serviceBrokerAsset := assets.NewAssets().ServiceBroker
	serviceBrokerAppName := PushApp(serviceBrokerAsset, config.RubyBuildpackName)

	serviceName := initiateBrokerConfig(serviceBrokerAppName)

	brokerName := generator.PrefixedRandomName("RATS-BROKER-")
	brokerUrl := helpers.AppUri(serviceBrokerAppName, "")

	config = helpers.LoadConfig()
	context := helpers.NewContext(config)
	cf.AsUser(context.AdminUserContext(), context.ShortTimeout(), func() {
		session := cf.Cf("create-service-broker", brokerName, "user", "password", brokerUrl)
		Expect(session.Wait(DEFAULT_TIMEOUT)).To(Exit(0))

		session = cf.Cf("enable-service-access", serviceName)
		Expect(session.Wait(DEFAULT_TIMEOUT)).To(Exit(0))

	})

	return brokerName, serviceBrokerAppName, serviceName
}

func initiateBrokerConfig(serviceBrokerAppName string) string {
	brokerConfigJson := helpers.CurlApp(serviceBrokerAppName, "/config")

	var brokerConfigMap customMap

	err := json.Unmarshal([]byte(brokerConfigJson), &brokerConfigMap)
	Expect(err).NotTo(HaveOccurred())

	dashboardClientId := generator.PrefixedRandomName("RATS-DASHBOARD-ID-")
	serviceName := generator.PrefixedRandomName("RATS-SERVICE-")
	serviceId := generator.PrefixedRandomName("RATS-SERVICE-ID-")

	services := brokerConfigMap.key("behaviors").key("catalog").key("body")["services"].([]interface{})
	service := services[0].(map[string]interface{})
	service["dashboard_client"].(map[string]interface{})["id"] = dashboardClientId
	service["name"] = serviceName
	service["id"] = serviceId

	plans := service["plans"].([]interface{})

	for i, plan := range plans {
		servicePlanId := generator.PrefixedRandomName(fmt.Sprintf("RATS-SERVICE-PLAN-ID-%d-", i))
		plan.(map[string]interface{})["id"] = servicePlanId
	}

	changedJson, err := json.Marshal(brokerConfigMap)
	Expect(err).NotTo(HaveOccurred())

	helpers.CurlApp(serviceBrokerAppName, "/config", "-X", "POST", "-d", string(changedJson))

	return serviceName
}
