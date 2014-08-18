package helpers

import (
	"encoding/json"
	"fmt"
	"time"

	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers"
)

const (
	BROKER_START_TIMEOUT = 5 * time.Minute
	DEFAULT_TIMEOUT      = 30 * time.Second
)

type ServiceBroker struct {
	Name    string
	Path    string
	context helpers.SuiteContext
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

type ServiceInstance struct {
	Metadata struct {
		Guid string `json:"guid"`
	}
}

type ServiceInstanceResponse struct {
	Resources []ServiceInstance
}

type SpaceJson struct {
	Resources []struct {
		Metadata struct {
			Guid string
		}
	}
}

func NewServiceBroker(name string, path string, context helpers.SuiteContext) ServiceBroker {
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
	b.context = context
	return b
}

func (b ServiceBroker) Push() {
	Expect(cf.Cf("push", b.Name, "-p", b.Path).Wait(BROKER_START_TIMEOUT)).To(Exit(0))
}

func (b ServiceBroker) Configure() {
	Expect(cf.Cf("set-env", b.Name, "CONFIG", b.ToJSON()).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	b.Restart()
}

func (b ServiceBroker) Restart() {
	Expect(cf.Cf("restart", b.Name).Wait(BROKER_START_TIMEOUT)).To(Exit(0))
}

func (b ServiceBroker) Create() {
	cf.AsUser(b.context.AdminUserContext(), func() {
		Expect(cf.Cf("create-service-broker", b.Name, "username", "password", helpers.AppUri(b.Name, "")).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		Expect(cf.Cf("service-brokers").Wait(DEFAULT_TIMEOUT)).To(Say(b.Name))
	})
}

func (b ServiceBroker) Update() {
	cf.AsUser(b.context.AdminUserContext(), func() {
		Expect(cf.Cf("update-service-broker", b.Name).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	})
}

func (b ServiceBroker) Delete() {
	cf.AsUser(b.context.AdminUserContext(), func() {
		Expect(cf.Cf("delete-service-broker", b.Name, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))

		brokers := cf.Cf("service-brokers").Wait(DEFAULT_TIMEOUT)
		Expect(brokers).To(Exit(0))
		Expect(brokers.Out.Contents()).ToNot(ContainSubstring(b.Name))
	})
}

func (b ServiceBroker) Destroy() {
	cf.AsUser(b.context.AdminUserContext(), func() {
		Expect(cf.Cf("purge-service-offering", b.Service.Name, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	})
	b.Delete()
	Expect(cf.Cf("delete", b.Name, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
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
	var session *Session
	cf.AsUser(b.context.AdminUserContext(), func() {
		session = cf.Cf("curl", url).Wait(DEFAULT_TIMEOUT)
		Expect(session).To(Exit(0))
	})
	structure := ServicesResponse{}
	json.Unmarshal(session.Out.Contents(), &structure)

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
	cf.AsUser(b.context.AdminUserContext(), func() {
		Expect(cf.Cf("curl", url, "-X", "PUT", "-d", string(planJson)).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	})
}

func (b ServiceBroker) CreateServiceInstance(instanceName string) string {
	Expect(cf.Cf("create-service", b.Service.Name, b.Plan.Name, instanceName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	url := fmt.Sprintf("/v2/service_instances?q=name:%s", instanceName)
	serviceInstance := ServiceInstanceResponse{}
	curl := cf.Cf("curl", url).Wait(DEFAULT_TIMEOUT)
	Expect(curl).To(Exit(0))
	json.Unmarshal(curl.Out.Contents(), &serviceInstance)
	return serviceInstance.Resources[0].Metadata.Guid
}

func (b ServiceBroker) GetSpaceGuid() string {
	url := fmt.Sprintf("/v2/spaces?q=name%%3A%s", b.context.RegularUserContext().Space)
	jsonResults := SpaceJson{}
	curl := cf.Cf("curl", url).Wait(DEFAULT_TIMEOUT)
	Expect(curl).To(Exit(0))
	json.Unmarshal(curl.Out.Contents(), &jsonResults)
	return jsonResults.Resources[0].Metadata.Guid
}
