package services

import (
	"os"
	"io/ioutil"
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/vito/cmdtest/matchers"

	. "github.com/pivotal-cf-experimental/cf-test-helpers/generator"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
	. "github.com/pivotal-cf-experimental/cf-acceptance-tests/helpers"
)

type ServicesResponse struct {
	Resources []ServiceResponse
}

type ServiceResponse struct {
	Entity struct {
		Label string
		ServicePlans []ServicePlanResponse `json:"service_plans"`
	}
}

type ServicePlanResponse struct {
	Entity struct {
		Name string
		Public bool
	}
	Metadata struct {
		Url string
	}
}

var _ = Describe("Service Broker Lifecycle", func() {
	var appName string

	BeforeEach(func() {
		appName = RandomName()
		Cf("login", "-u", os.Getenv("ADMIN_USER"), "-p", os.Getenv("ADMIN_PASSWORD"))
		Expect(Cf("push", appName, "-p", serviceBrokerPath)).To(Say("App started"))
		configJSON, _ := ioutil.ReadFile(ServiceBrokerConfigPath)
		Expect(Cf("set-env", appName, "CONFIG", string(configJSON))).To(ExitWithTimeout(0, 2*time.Second))
		Expect(Cf("restart", appName)).To(Say("App started"))
	})

	AfterEach(func() {
		Expect(Cf("delete-service-broker", appName, "-f")).To(ExitWithTimeout(0, 2*time.Second))
		Expect(Cf("delete", appName, "-f")).To(ExitWithTimeout(0, 2*time.Second))
		Cf("login", "-u", os.Getenv("CF_USER"), "-p", os.Getenv("CF_USER_PASSWORD"))
	})

	It("confirms correct behavior in the lifecycle of a service broker", func() {
		defer Recover() // Catches panic thrown by Require expectations

		// Adding the service broker
		Require(Cf("create-service-broker", appName, "username", "password", AppUri(appName, ""))).To(ExitWithTimeout(0, 2*time.Second))
		Expect(Cf("service-brokers")).To(Say(appName))

		// Confirming the plans are not yet public
		session := Cf("marketplace")
		Expect(session).NotTo(Say(ServiceBrokerConfig.FirstBrokerServiceLabel))
		Expect(session).NotTo(Say(ServiceBrokerConfig.FirstBrokerPlanName))

		// Making the plans public
		session = Cf("curl", "/v2/services?inline-relations-depth=1")
		structure := ServicesResponse{}
		json.Unmarshal(session.FullOutput(), &structure)
		for _, service := range structure.Resources {
			if service.Entity.Label == ServiceBrokerConfig.FirstBrokerServiceLabel {
				for _, plan := range service.Entity.ServicePlans {
					if plan.Entity.Name == ServiceBrokerConfig.FirstBrokerPlanName {
						MakePlanPublic(plan.Metadata.Url)
						break
					}
				}
			}
		}

		// Confirming plans show up in the marketplace
		session = Cf("marketplace")
		Expect(session).To(Say(ServiceBrokerConfig.FirstBrokerServiceLabel))
		Expect(session).To(Say(ServiceBrokerConfig.FirstBrokerPlanName))

		// Changing the catalog on the broker
		Eventually(Curling(AppUri(appName,"/v2/catalog"), "-X", "POST", "-i")).Should(Say("HTTP/1.1 200 OK"))
		Require(Cf("update-service-broker", appName, "username", "password", AppUri(appName, ""))).To(ExitWithTimeout(0, 2*time.Second))

		// Confirming the changes to the broker show up in the marketplace
		session = Cf("marketplace")
		Expect(session).NotTo(Say(ServiceBrokerConfig.FirstBrokerServiceLabel))
		Expect(session).NotTo(Say(ServiceBrokerConfig.FirstBrokerPlanName))
		Expect(session).To(Say(ServiceBrokerConfig.SecondBrokerServiceLabel))
		Expect(session).To(Say(ServiceBrokerConfig.SecondBrokerPlanName))

		// Deleting the service broker and confirming the plans no longer display
		Require(Cf("delete-service-broker", appName, "-f")).To(ExitWithTimeout(0, 2*time.Second))
		session = Cf("marketplace")
		Expect(session).NotTo(Say(ServiceBrokerConfig.FirstBrokerServiceLabel))
		Expect(session).NotTo(Say(ServiceBrokerConfig.FirstBrokerPlanName))
		Expect(session).NotTo(Say(ServiceBrokerConfig.SecondBrokerServiceLabel))
		Expect(session).NotTo(Say(ServiceBrokerConfig.SecondBrokerPlanName))
	})
})
