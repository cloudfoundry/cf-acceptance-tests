package routing

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"

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

var _ = Describe(deaUnsupportedTag+"Route Services", func() {
	config := helpers.LoadConfig()

	if config.IncludeRouteServices {
		Context("when a route binds to a service", func() {
			Context("when service broker returns a route service url", func() {
				var (
					brokerName               string
					serviceInstanceName      string
					appName                  string
					routeServiceName         string
					golangAsset              = assets.NewAssets().Golang
					loggingRouteServiceAsset = assets.NewAssets().LoggingRouteServiceZip
				)

				BeforeEach(func() {
					brokerAppName := GenerateAppName()
					brokerName := generator.PrefixedRandomName("RATS-BROKER-")
					serviceName := generator.PrefixedRandomName("RATS-SERVICE-")

					createServiceBroker(brokerName, brokerAppName, serviceName)
					serviceInstanceName := generator.PrefixedRandomName("RATS-SERVICE-")
					createServiceInstance(serviceInstanceName, serviceName)

					appName = GenerateAppName()
					PushAppNoStart(appName, golangAsset, config.GoBuildpackName)
					app_helpers.EnableDiego(appName)
					StartApp(appName)

					routeServiceName = GenerateAppName()
					PushApp(routeServiceName, loggingRouteServiceAsset, config.GoBuildpackName)
					configureBroker(brokerAppName, routeServiceName)

					bindRouteToService(appName, serviceInstanceName)
				})

				AfterEach(func() {
					app_helpers.AppReport(appName, DEFAULT_TIMEOUT)

					unbindRouteFromService(appName, serviceInstanceName)
					deleteServiceInstance(serviceInstanceName)
					deleteServiceBroker(brokerName)
					DeleteApp(appName)
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
					serviceInstanceName string
					appName             string
					golangAsset         = assets.NewAssets().Golang
				)

				BeforeEach(func() {
					brokerAppName := GenerateAppName()
					brokerName := generator.PrefixedRandomName("RATS-BROKER-")
					serviceName := generator.PrefixedRandomName("RATS-SERVICE-")

					createServiceBroker(brokerName, brokerAppName, serviceName)
					serviceInstanceName := generator.PrefixedRandomName("RATS-SERVICE-")
					createServiceInstance(serviceInstanceName, serviceName)

					appName = GenerateAppName()
					PushAppNoStart(appName, golangAsset, config.GoBuildpackName)
					app_helpers.EnableDiego(appName)
					StartApp(appName)

					configureBroker(brokerAppName, "")

					bindRouteToService(appName, serviceInstanceName)
				})

				AfterEach(func() {
					app_helpers.AppReport(appName, DEFAULT_TIMEOUT)

					unbindRouteFromService(appName, serviceInstanceName)
					deleteServiceInstance(serviceInstanceName)
					deleteServiceBroker(brokerName)
					DeleteApp(appName)
				})

				It("routes to an app", func() {
					Eventually(func() string {
						return helpers.CurlAppRoot(appName)
					}, DEFAULT_TIMEOUT).Should(ContainSubstring("go, world"))
				})
			})

			Context("when arbitrary parameters are sent", func() {
				var (
					brokerName          string
					brokerAppName       string
					serviceInstanceName string
					domain              string
					hostname            string
				)

				BeforeEach(func() {
					domain = config.AppsDomain
					spacename := context.RegularUserContext().Space
					hostname = generator.PrefixedRandomName("RATS-HOSTNAME-")

					brokerAppName := GenerateAppName()
					brokerName := generator.PrefixedRandomName("RATS-BROKER-")
					serviceName := generator.PrefixedRandomName("RATS-SERVICE-")

					createServiceBroker(brokerName, brokerAppName, serviceName)
					serviceInstanceName := generator.PrefixedRandomName("RATS-SERVICE-")
					createServiceInstance(serviceInstanceName, serviceName)

					createRoute(hostname, "", spacename, domain)

					configureBroker(brokerAppName, "")
				})

				AfterEach(func() {
					unbindRouteFromService(hostname, serviceInstanceName)
					deleteServiceInstance(serviceInstanceName)
					deleteServiceBroker(brokerName)
					DeleteRoute(hostname, "", domain)
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
	}
})

type customMap map[string]interface{}

func (c customMap) key(key string) customMap {
	return c[key].(map[string]interface{})
}

func bindRouteToService(hostname, serviceInstanceName string) {
	routeGuid := getRouteGuid(hostname, "")
	serviceInstanceGuid := getServiceInstanceGuid(serviceInstanceName)
	cf.Cf("curl", fmt.Sprintf("/v2/service_instances/%s/routes/%s", serviceInstanceGuid, routeGuid), "-X", "PUT")

	Eventually(func() string {
		response := cf.Cf("curl", fmt.Sprintf("/v2/routes/%s", routeGuid))
		Expect(response.Wait(DEFAULT_TIMEOUT)).To(Exit(0))

		contents := response.Out.Contents()
		return string(contents)
	}, DEFAULT_TIMEOUT, "1s").ShouldNot(ContainSubstring(`"service_instance_guid": null`))
}

func bindRouteToServiceWithParams(hostname, serviceInstanceName string, params string) {
	routeGuid := getRouteGuid(hostname, "")
	serviceInstanceGuid := getServiceInstanceGuid(serviceInstanceName)
	cf.Cf("curl", fmt.Sprintf("/v2/service_instances/%s/routes/%s", serviceInstanceGuid, routeGuid), "-X", "PUT",
		"-d", fmt.Sprintf("{\"parameters\": %s}", params))

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

func unbindRouteFromService(hostname, serviceInstanceName string) {
	routeGuid := getRouteGuid(hostname, "")
	guid := getServiceInstanceGuid(serviceInstanceName)
	cf.Cf("curl", fmt.Sprintf("/v2/service_instances/%s/routes/%s", guid, routeGuid), "-X", "DELETE")

	Eventually(func() string {
		response := cf.Cf("curl", fmt.Sprintf("/v2/routes/%s", routeGuid))
		Expect(response.Wait(DEFAULT_TIMEOUT)).To(Exit(0))

		contents := response.Out.Contents()
		return string(contents)
	}, DEFAULT_TIMEOUT, "1s").Should(ContainSubstring(`"service_instance_guid": null`))
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

func createServiceInstance(serviceInstanceName, serviceName string) {
	session := cf.Cf("create-service", serviceName, "fake-plan", serviceInstanceName)
	Expect(session.Wait(DEFAULT_TIMEOUT)).To(Exit(0))
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
	PushApp(brokerAppName, serviceBrokerAsset, config.RubyBuildpackName)

	initiateBrokerConfig(serviceName, brokerAppName)

	brokerUrl := helpers.AppUri(brokerAppName, "")

	config = helpers.LoadConfig()
	context := helpers.NewContext(config)
	cf.AsUser(context.AdminUserContext(), context.ShortTimeout(), func() {
		session := cf.Cf("create-service-broker", brokerName, "user", "password", brokerUrl)
		Expect(session.Wait(DEFAULT_TIMEOUT)).To(Exit(0))

		session = cf.Cf("enable-service-access", serviceName)
		Expect(session.Wait(DEFAULT_TIMEOUT)).To(Exit(0))

	})
}

func initiateBrokerConfig(serviceName, serviceBrokerAppName string) {
	brokerConfigJson := helpers.CurlApp(serviceBrokerAppName, "/config")

	var brokerConfigMap customMap

	err := json.Unmarshal([]byte(brokerConfigJson), &brokerConfigMap)
	Expect(err).NotTo(HaveOccurred())

	dashboardClientId := generator.PrefixedRandomName("RATS-DASHBOARD-ID-")
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
}

func targetedSpaceGuid() string {
	output := cf.Cf("target")
	Expect(output.Wait()).To(Exit(0))

	targetInfo := strings.TrimSpace(string(output.Out.Contents()))
	spaceMatch, _ := regexp.Compile(`Space:\s+([^\s]+)`)
	spaceName := spaceMatch.FindAllStringSubmatch(targetInfo, -1)[0][1]

	output = cf.Cf("space", spaceName, "--guid")
	Expect(output.Wait()).To(Exit(0))

	return strings.TrimSpace(string(output.Out.Contents()))
}

func sharedDomainGuid() string {
	output := cf.Cf("curl", "/v2/shared_domains")
	Expect(output.Wait()).To(Exit(0))

	var sharedDomainMap response

	err := json.Unmarshal(output.Out.Contents(), &sharedDomainMap)
	Expect(err).NotTo(HaveOccurred())

	return sharedDomainMap.Resources[0].Metadata.Guid
}
