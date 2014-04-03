package helpers

import (
	"encoding/json"
	"fmt"
	"time"

	. "github.com/cloudfoundry/cf-acceptance-tests/helpers"
	. "github.com/onsi/gomega"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
	"github.com/pivotal-cf-experimental/cf-test-helpers/generator"
	"github.com/vito/cmdtest"
	. "github.com/vito/cmdtest/matchers"
)

type ServiceBroker struct {
	Name    string
	Path    string
	Service struct {
		Name            string `json:"name"`
		ID              string `json:"id"`
		DashboardClient struct {
			ID          string `json:"id"`
			Secret      string `json:"secret"`
			RedirectUri string `json:"redirect_uri"`
		}
	}
	Plan struct {
		Name string `json:"name"`
		ID   string `json:"id"`
	}
}

type ServicesResponse struct {
	Resources []ServiceResponse
}

type ServiceResponse struct {
	Entity struct {
		Label        string
		ServicePlans []ServicePlanResponse `json:"service_plans"`
	}
}

type ServicePlanResponse struct {
	Entity struct {
		Name   string
		Public bool
	}
	Metadata struct {
		Url  string
		Guid string
	}
}

type ServiceInstanceResponse struct {
	Metadata struct {
		Guid string `json:"guid"`
	}
}

func NewServiceBroker(name string, path string) ServiceBroker {
	b := ServiceBroker{}
	b.Path = path
	b.Name = name
	b.Service.Name = generator.RandomName()
	b.Service.ID = generator.RandomName()
	b.Plan.Name = generator.RandomName()
	b.Plan.ID = generator.RandomName()
	b.Service.DashboardClient.ID = generator.RandomName()
	b.Service.DashboardClient.Secret = generator.RandomName()
	b.Service.DashboardClient.RedirectUri = generator.RandomName()
	return b
}

var appStartTimeout = 3 * 60 * time.Second

func (b ServiceBroker) Push() {
	Expect(Cf("push", b.Name, "-p", b.Path)).To(ExitWithTimeout(0, appStartTimeout))
}

func (b ServiceBroker) Configure() {
	Expect(Cf("set-env", b.Name, "CONFIG", b.ToJSON())).To(ExitWithTimeout(0, 2*time.Second))
	b.Restart()
}

func (b ServiceBroker) Restart() {
	Expect(Cf("restart", b.Name)).To(ExitWithTimeout(0, appStartTimeout))
}

func (b ServiceBroker) Create(appsDomain string) {
	AsUser(AdminUserContext, func() {
		Require(Cf("create-service-broker", b.Name, "username", "password", AppUri(b.Name, "", appsDomain))).To(ExitWithTimeout(0, 30*time.Second))
		Expect(Cf("service-brokers")).To(Say(b.Name))
	})
}

func (b ServiceBroker) Update(appsDomain string) {
	AsUser(AdminUserContext, func() {
		Require(Cf("update-service-broker", b.Name, "username", "password", AppUri(b.Name, "", appsDomain))).To(ExitWithTimeout(0, 30*time.Second))
	})
}

func (b ServiceBroker) Delete() {
	AsUser(AdminUserContext, func() {
		Expect(Cf("delete-service-broker", b.Name, "-f")).To(ExitWithTimeout(0, 10*time.Second))
	})
	Expect(Cf("service-brokers")).ToNot(Say(b.Name))
}

func (b ServiceBroker) Destroy() {
	AsUser(AdminUserContext, func() {
		Expect(Cf("purge-service-offering", b.Service.Name, "-f")).To(ExitWithTimeout(0, 10*time.Second))
	})
	b.Delete()
	Expect(Cf("delete", b.Name, "-f")).To(ExitWithTimeout(0, 10*time.Second))
}

func (b ServiceBroker) ToJSON() string {
	attributes := make(map[string]interface{})
	attributes["service"] = b.Service
	attributes["plan"] = b.Plan
	attributes["dashboard_client"] = b.Service.DashboardClient
	jsonBytes, _ := json.Marshal(attributes)
	return string(jsonBytes)
}

func (b ServiceBroker) PublicizePlans() {
	url := fmt.Sprintf("/v2/services?inline-relations-depth=1&q=label:%s", b.Service.Name)
	var session *cmdtest.Session
	AsUser(AdminUserContext, func() {
		session = Cf("curl", url)
	})
	structure := ServicesResponse{}
	json.Unmarshal(session.FullOutput(), &structure)
	for _, service := range structure.Resources {
		if service.Entity.Label == b.Service.Name {
			for _, plan := range service.Entity.ServicePlans {
				if plan.Entity.Name == b.Plan.Name {
					b.PublicizePlan(plan.Metadata.Url)
					break
				}
			}
		}
	}
}

func (b ServiceBroker) PublicizePlan(url string) {
	jsonMap := make(map[string]bool)
	jsonMap["public"] = true
	planJson, _ := json.Marshal(jsonMap)
	AsUser(AdminUserContext, func() {
		Expect(Cf("curl", url, "-X", "PUT", "-d", string(planJson))).To(ExitWithTimeout(0, 5*time.Second))
	})
}

func (b ServiceBroker) CreateServiceInstance(instanceName string) (guid string) {
	// TODO:  CreateServiceInstance is used as a workaround for the problem in cf 6.0.1 that prevents us from
	//        creating an instance of a service when there are more than 50 services in the environment.
	//        Should be replaced by the following line ASAP

	// Expect(Cf("create-service", broker.Service.Name, broker.Plan.Name, instanceName)).To(ExitWith(0))
	url := fmt.Sprintf("/v2/services?inline-relations-depth=1&q=label:%s", b.Service.Name)
	session := Cf("curl", url)
	structure := ServicesResponse{}
	json.Unmarshal(session.FullOutput(), &structure)
	for _, service := range structure.Resources {
		if service.Entity.Label == b.Service.Name {
			for _, plan := range service.Entity.ServicePlans {
				if plan.Entity.Name == b.Plan.Name {
					guid = b.createInstanceForPlan(plan.Metadata.Guid, instanceName)
					break
				}
			}
		}
	}
	return
}

func (b ServiceBroker) createInstanceForPlan(planGuid, instanceName string) (guid string) {
	spaceGuid := b.GetSpaceGuid()

	attributes := make(map[string]string)
	attributes["name"] = instanceName
	attributes["service_plan_guid"] = planGuid
	attributes["space_guid"] = spaceGuid
	jsonBytes, _ := json.Marshal(attributes)

	result := Cf("curl", "/v2/service_instances", "-X", "POST", "-d", string(jsonBytes))
	Expect(result).To(ExitWith(0))

	serviceInstance := ServiceInstanceResponse{}
	json.Unmarshal(result.FullOutput(), &serviceInstance)

	guid = serviceInstance.Metadata.Guid
	return
}

type SpaceJson struct {
	Resources []struct {
		Metadata struct {
			Guid string
		}
	}
}

func (b ServiceBroker) GetSpaceGuid() string {
	url := fmt.Sprintf("/v2/spaces?q=name%%3A%s", RegularUserContext.Space)
	session := Cf("curl", url)
	jsonResults := SpaceJson{}
	json.Unmarshal(session.FullOutput(), &jsonResults)
	return jsonResults.Resources[0].Metadata.Guid
}
