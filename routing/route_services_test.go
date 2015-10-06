package routing

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Route Services", func() {
	Context("when a route binds a service", func() {
		Context("when service broker does not return a route service url", func() {
			var (
				appName     string
				golangAsset = assets.NewAssets().Golang
			)

			BeforeEach(func() {
				appName := PushApp(golangAsset, config.GoBuildpackName)
				EnableDiego(appName)

				routeServiceName = PushApp(loggingRouteServiceAsset, config.GoBuildpackName)

				configureBroker(brokerAppName, "")

				bindRouteToService(appName, serviceInstanceName)
				RestartApp(appName)
			})

			It("routes to an app", func() {
				Eventually(func() string {
					return helpers.CurlAppRoot(appName)
				}, DEFAULT_TIMEOUT).Should(ContainSubstring("go, world"))
			})
		})

		Context("when service broker returns a route service url", func() {
			var (
				appName                  string
				routeServiceName         string
				golangAsset              = assets.NewAssets().Golang
				loggingRouteServiceAsset = assets.NewAssets().LoggingRouteServiceZip
			)

			BeforeEach(func() {
				appName = PushApp(golangAsset)
				EnableDiego(appName)

				routeServiceName = PushApp(loggingRouteServiceAsset)
				configureBroker(brokerAppName, routeServiceName)

				bindRouteToService(appName, serviceInstanceName)
				RestartApp(appName)
			})

			AfterEach(func() {
				route_guid := getRouteGuid(appName)
				guid := getServiceInstanceGuid(serviceInstanceName)
				cf.Cf("curl", fmt.Sprintf("/v2/service_instances/%s/routes/%s", guid, route_guid), "-X", "DELETE")

				Eventually(func() string {
					response := cf.Cf("curl", fmt.Sprintf("/v2/routes/%s", route_guid))
					Expect(response.Wait(DEFAULT_TIMEOUT)).To(Exit(0))

					contents := response.Out.Contents()
					return string(contents)
				}, DEFAULT_TIMEOUT, "1s").Should(ContainSubstring(`"service_instance_guid": null`))

			})

			It("a request to the app is routed through the route service", func() {
				Eventually(func() string {
					return helpers.CurlAppRoot(appName)
				}, DEFAULT_TIMEOUT).Should(ContainSubstring("go, world"))

				Eventually(func() *Session {
					logs := cf.Cf("logs", "--recent", routeServiceName)
					Expect(logs.Wait(DEFAULT_TIMEOUT)).To(Exit(0))
					return logs
				}, DEFAULT_TIMEOUT).Should(Say("Response Body: go, world"))
			})
		})
	})
})

type customMap map[string]interface{}

func (c customMap) key(key string) customMap {
	return c[key].(map[string]interface{})
}

func bindRouteToService(routeName string, serviceInstanceName string) {
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

func createServiceInstance() string {
	serviceInstanceName := generator.PrefixedRandomName("RATS-SERVICE-")

	// create service instance
	session := cf.Cf("create-service", "fake-service", "fake-plan", serviceInstanceName)
	Expect(session.Wait(DEFAULT_TIMEOUT)).To(Exit(0))

	return serviceInstanceName
}

func configureBroker(serviceBrokerAppName, routeServiceName string) {
	// downloadServiceBrokerJsonConfig
	brokerConfigJson := helpers.CurlApp(serviceBrokerAppName, "/config")

	var brokerConfigMap customMap

	err := json.Unmarshal([]byte(brokerConfigJson), &brokerConfigMap)
	Expect(err).NotTo(HaveOccurred())

	// updateConfigWithOurCustomRoute
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

	// uploadNewServiceBrokerConfig
	helpers.CurlApp(serviceBrokerAppName, "/config", "-X", "POST", "-d", string(changedJson))
}
