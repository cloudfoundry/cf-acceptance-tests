package helpers

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	. "github.com/cloudfoundry/cf-acceptance-tests/helpers"
	. "github.com/onsi/gomega"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
	"github.com/pivotal-cf-experimental/cf-test-helpers/generator"
	. "github.com/vito/cmdtest/matchers"
)

type ServiceBroker struct {
	Name    string
	Path    string
	Service struct {
		Name string `json:"name"`
		ID   string `json:"id"`
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

func NewServiceBroker(name string) ServiceBroker {
	b := ServiceBroker{}
	b.Path, _ = filepath.Abs("../assets/service_broker/")
	b.Name = name
	b.Service.Name = generator.RandomName()
	b.Service.ID = generator.RandomName()
	b.Plan.Name = generator.RandomName()
	b.Plan.ID = generator.RandomName()
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
	Require(Cf("create-service-broker", b.Name, "username", "password", AppUri(b.Name, "", appsDomain))).To(ExitWithTimeout(0, 30*time.Second))
	Expect(Cf("service-brokers")).To(Say(b.Name))
}

func (b ServiceBroker) Destroy() {
	Expect(Cf("delete-service-broker", b.Name, "-f")).To(ExitWithTimeout(0, 2*time.Second))
	Expect(Cf("delete", b.Name, "-f")).To(ExitWithTimeout(0, 2*time.Second))
}

func (b ServiceBroker) ToJSON() string {
	attributes := make(map[string]interface{})
	attributes["service"] = b.Service
	attributes["plan"] = b.Plan
	jsonBytes, _ := json.Marshal(attributes)
	return string(jsonBytes)
}

func (b ServiceBroker) PublicizePlans() {
	url := fmt.Sprintf("/v2/services?inline-relations-depth=1&q=label:%s", b.Service.Name)
	session := Cf("curl", url)
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
	Expect(Cf("curl", url, "-X", "PUT", "-d", string(planJson))).To(ExitWithTimeout(0, 5*time.Second))
}

func (b ServiceBroker) CreateServiceInstance(instanceName string) {
	url := fmt.Sprintf("/v2/services?inline-relations-depth=1&q=label:%s", b.Service.Name)
	session := Cf("curl", url)
	structure := ServicesResponse{}
	json.Unmarshal(session.FullOutput(), &structure)
	for _, service := range structure.Resources {
		if service.Entity.Label == b.Service.Name {
			for _, plan := range service.Entity.ServicePlans {
				if plan.Entity.Name == b.Plan.Name {
					b.createInstanceForPlan(plan.Metadata.Guid, instanceName)
					break
				}
			}
		}
	}
}

func (b ServiceBroker) createInstanceForPlan(planGuid, instanceName string) {
	spaceGuid := b.GetSpaceGuid()

	attributes := make(map[string]string)
	attributes["name"] = instanceName
	attributes["service_plan_guid"] = planGuid
	attributes["space_guid"] = spaceGuid
	jsonBytes, _ := json.Marshal(attributes)
	Expect(Cf("curl", "/v2/service_instances", "-X", "POST", "-d", string(jsonBytes))).To(ExitWith(0))
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
