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
	Context("when an app has a route service bound", func() {
		var (
			appName                  string
			routeServiceName         string
			golangAsset              = assets.NewAssets().Golang
			loggingRouteServiceAsset = assets.NewAssets().LoggingRouteServiceZip
			brokerName               string
			serviceInstanceName      string
		)

		BeforeEach(func() {
			appName := PushApp(golangAsset, config.GoBuildpackName)
			EnableDiego(appName)

			routeServiceName = PushApp(loggingRouteServiceAsset, config.GoBuildpackName)

			serviceInstanceName, brokerName = createServiceBrokerAndServiceInstance(routeServiceName)

			// create service broker
			//
			// config routeservice url
			//
			// creaste service instance

			bindRouteToService(appName, serviceInstanceName)

			RestartApp(appName)
		})

		AfterEach(func() {
			config = helpers.LoadConfig()
			context := helpers.NewContext(config)

			cf.AsUser(context.AdminUserContext(), context.ShortTimeout(), func() {
				guid := getServiceInstanceGuid(serviceInstanceName)
				route_guid := getRouteGuid(appName)
				cf.Cf("curl", fmt.Sprintf("/v2/service_instances/%s/routes/%s", guid, route_guid), "-X", "DELETE")

				Eventually(func() string {
					response := cf.Cf("curl", fmt.Sprintf("/v2/routes/%s", route_guid))
					Expect(response.Wait(DEFAULT_TIMEOUT)).To(Exit(0))

					contents := response.Out.Contents()
					return string(contents)
				}, DEFAULT_TIMEOUT, "1s").Should(ContainSubstring(`"service_instance_guid": null`))

				Eventually(func() string {
					response := cf.Cf("curl", fmt.Sprintf("/v2/service_instances/%s", guid), "-X", "DELETE")
					Expect(response.Wait(DEFAULT_TIMEOUT)).To(Exit(0))

					contents := response.Out.Contents()
					return string(contents)
				}, DEFAULT_TIMEOUT, "1s").Should(ContainSubstring("CF-ServiceInstanceNotFound"))

				responseBuffer := cf.Cf("delete-service-broker", brokerName, "-f")
				Expect(responseBuffer.Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			})
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

type customMap map[string]interface{}

func (c customMap) key(key string) customMap {
	return c[key].(map[string]interface{})
}

func bindRouteToService(routeName string, serviceInstanceName string) {
	// PUT /v2/service_instances/b512b03f-0445-4ffc-a8cf-1c16a29b33a0/routes/28928926-23db-4a90-a44c-5e6ff5b831cc
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

func createServiceBrokerAndServiceInstance(routeServiceName string) (string, string) {
	serviceBrokerAsset := assets.NewAssets().ServiceBroker
	serviceBrokerAppName := PushApp(serviceBrokerAsset)

	// downloadServiceBrokerJsonConfig
	brokerConfigJson := helpers.CurlApp(serviceBrokerAppName, "/config")

	var brokerConfigMap customMap

	err := json.Unmarshal([]byte(brokerConfigJson), &brokerConfigMap)
	Expect(err).NotTo(HaveOccurred())

	// updateConfigWithOurCustomRoute
	routeServiceUrl := helpers.AppUri(routeServiceName, "/")
	url, err := url.Parse(routeServiceUrl)
	Expect(err).NotTo(HaveOccurred())
	url.Scheme = "https"
	routeServiceUrl = url.String()

	brokerConfigMap.key("behaviors").key("bind").key("default").key("body")["route_service_url"] = routeServiceUrl

	changedJson, err := json.Marshal(brokerConfigMap)
	Expect(err).NotTo(HaveOccurred())

	// uploadNewServiceBrokerConfig
	helpers.CurlApp(serviceBrokerAppName, "/config", "-X", "POST", "-d", string(changedJson))

	brokerUrl := helpers.AppUri(serviceBrokerAppName, "")

	brokerName := generator.PrefixedRandomName("RATS-BROKER-")

	config = helpers.LoadConfig()
	context := helpers.NewContext(config)

	cf.AsUser(context.AdminUserContext(), context.ShortTimeout(), func() {
		session := cf.Cf("target")
		Expect(session.Wait(DEFAULT_TIMEOUT)).To(Exit(0))

		// registerAsBroker
		// cf create-service-broker name user password url
		session = cf.Cf("create-service-broker", brokerName, "user", "password", brokerUrl)
		Expect(session.Wait(DEFAULT_TIMEOUT)).To(Exit(0))

		// public service access
		// cf enable-service-access name
		session = cf.Cf("enable-service-access", "fake-service")
		Expect(session.Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	})

	serviceInstanceName := "our-fake-service"

	// create service instance
	session := cf.Cf("create-service", "fake-service", "fake-plan", serviceInstanceName)
	Expect(session.Wait(DEFAULT_TIMEOUT)).To(Exit(0))

	return serviceInstanceName, brokerName
}

// func createServiceBroker(routeServiceName string) {
// 	serviceBrokerAsset := assets.NewAssets().ServiceBroker
// 	serviceBrokerAppName := PushApp(serviceBrokerAsset)
